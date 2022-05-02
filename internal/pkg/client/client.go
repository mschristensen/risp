package client

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"
	"risp/internal"
	"risp/internal/pkg/log"
	"risp/internal/pkg/session"
	"risp/pkg/checksum"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var logger logrus.FieldLogger = logrus.StandardLogger()

// DefaultWindowSize is the default initial window size for the client.
const DefaultWindowSize = 1 << 2

// MaxWindowSize is the maximum window size for the client.
const MaxWindowSize = 1 << 8

// Client implements the client behaviour of RISP.
type Client struct {
	serverAddr string
	uuid       uuid.UUID
	session    session.Session

	started        bool
	closing        bool
	done           bool
	lastWindowSize uint16
	checksum       *uint64

	conn    *grpc.ClientConn
	channel risppb.RISP_ConnectClient
}

// Cfg configures a Client.
type Cfg func(*Client) error

// WithServerPort sets the server port to connect to.
func WithServerPort(p uint16) Cfg {
	return func(c *Client) error {
		c.serverAddr = fmt.Sprintf("localhost:%d", p)
		return nil
	}
}

// WithSequenceLength sets the length of the sequence.
func WithSequenceLength(l uint16) Cfg {
	return func(c *Client) error {
		c.session.Sequence = make([]*uint32, l)
		return nil
	}
}

// WithRandomSequenceLength sets the sequence length to a random non-zero value in the supported range.
func WithRandomSequenceLength() Cfg {
	return func(c *Client) error {
		c.session.Sequence = make([]*uint32, rand.Intn(math.MaxUint16)+1) // nolint: gosec // we don't need high security here
		return nil
	}
}

// NewClient creates a new Client with the given configuration.
func NewClient(cfgs ...Cfg) (*Client, error) {
	client := &Client{}
	for _, cfg := range cfgs {
		if err := cfg(client); err != nil {
			return nil, errors.Wrap(err, "apply Client cfg failed")
		}
	}
	client.uuid = uuid.New()
	client.session.Window = DefaultWindowSize
	client.lastWindowSize = DefaultWindowSize
	return client, nil
}

// min returns the minimum of two values.
func min(a, b uint16) uint16 {
	if a < b {
		return a
	}
	return b
}

// sendRecv sends messages to the server that are received on the inbound channel,
// and receives messages from the server and sends them on the returned on the outbound channel.
func (c *Client) sendRecv(in chan *risppb.ClientMessage) (chan *risppb.ServerMessage, <-chan error) {
	out := make(chan *risppb.ServerMessage)
	kill := make(chan error)
	go func() {
		defer close(out)
		for {
			msg, err := c.channel.Recv()
			if err != nil {
				if c.done {
					return
				}
				kill <- errors.Wrap(err, "recv failed")
				return
			}
			out <- msg
		}
	}()
	go func() {
		for msg := range in {
			if err := c.channel.Send(msg); err != nil {
				kill <- errors.Wrap(err, "send failed")
				return
			}
		}
	}()
	return out, kill
}

// handleMessage updates the client state using the message from the server.
func (c *Client) handleMessage(_ context.Context, msg *risppb.ServerMessage) error {
	if msg.State == risppb.ConnectionState_CLOSING {
		if c.session.Ack != uint16(len(c.session.Sequence)) {
			return errors.New("received closing message before all items received")
		}
		c.closing = true
		sum, err := checksum.Sum(c.session.Sequence...)
		if err != nil {
			return errors.Wrap(err, "calculate checksum failed")
		}
		c.checksum = &sum
		return nil
	}
	if msg.State == risppb.ConnectionState_CLOSED {
		if c.session.Ack != uint16(len(c.session.Sequence)) {
			return errors.New("received closed message before all items received")
		}
		if c.checksum == nil {
			return errors.New("received closed message before checksum received")
		}
		c.done = true
		return nil
	}

	// store the item at the correct place in the sequence, as described by the offset
	c.session.Sequence[msg.Index] = &msg.Payload

	// update ack to reflect the index of the first missing value
	c.session.Ack = uint16(len(c.session.Sequence))
	for i := range c.session.Sequence {
		if c.session.Sequence[i] == nil {
			c.session.Ack = uint16(i)
			break
		}
	}

	// reduce the window size
	c.session.Window--

	return nil
}

// nextMessage prepares the next message to send to the server based on the current client state.
func (c *Client) nextMessage() *risppb.ClientMessage {
	msg := &risppb.ClientMessage{
		State: risppb.ConnectionState_CONNECTED,
		Uuid:  c.uuid[:],
		Len:   uint32(len(c.session.Sequence)),
	}

	msg.Window = uint32(c.session.Window)
	msg.Ack = uint32(c.session.Ack)

	if !c.started {
		msg.State = risppb.ConnectionState_CONNECTING
		c.started = true
		return msg
	}
	if c.closing && c.checksum != nil {
		msg.State = risppb.ConnectionState_CLOSED
		return msg
	}
	if c.checksum == nil && c.session.Ack == uint16(len(c.session.Sequence)) {
		msg.State = risppb.ConnectionState_CLOSING
		return msg
	}
	return msg
}

// Connect establishes the connection to the server.
func (c *Client) Connect(ctx context.Context) error {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil && status.Code(err) != codes.Canceled {
			return errors.Wrap(err, "close client connection failed")
		}
	}
	var err error
	c.conn, err = grpc.DialContext(ctx,
		c.serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO: use TLS
	)
	if err != nil {
		return errors.Wrapf(err, "connect to %s failed", c.serverAddr)
	}
	logger.Info("client connecting...")
	c.channel, err = risppb.NewRISPClient(c.conn).Connect(ctx)
	if err != nil {
		return errors.Wrap(err, "call connect failed")
	}
	return nil
}

// Run runs client-side RISP protocol to receive the integer stream from the server.
func (c *Client) Run(ctx context.Context) error {
	defer c.Reset()
	out := make(chan *risppb.ClientMessage)
	defer close(out)
	in, kill := c.sendRecv(out)

	// The client ticker is longer than the server ticker, so that we don't see duplicate messages.
	// Increasing this value can simulate what happens when messages arrive late from the server,
	// causing the client to retry messages.
	ticker := time.NewTicker(time.Duration(internal.ClientTickerMS) * time.Millisecond)
	defer ticker.Stop()
	killswitch := time.NewTicker(math.MaxInt64) // never ticks (for 290 years at least)
	if internal.ClientKillswitchMS > 0 {
		killswitch = time.NewTicker(time.Duration(internal.ClientKillswitchMS) * time.Millisecond)
	}
	defer killswitch.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-in:
			if !ok || msg == nil {
				return nil
			}
			logger.WithFields(log.ServerMessageToFields(msg)).Info("received message")
			if err := c.handleMessage(ctx, msg); err != nil {
				return errors.Wrap(err, "handle message failed")
			}
			if c.done {
				return nil
			}
		case <-ticker.C:
			if !c.started || c.session.Window == 0 || c.session.Ack == uint16(len(c.session.Sequence)) {
				if c.session.Window == 0 {
					c.session.Window = min(2*c.lastWindowSize, MaxWindowSize)
					c.lastWindowSize = c.session.Window
				}
				msg := c.nextMessage()
				out <- msg
				logger.WithFields(log.ClientMessageToFields(msg)).Info("sent message")
			}
		case <-killswitch.C:
			if err := c.channel.CloseSend(); err != nil {
				logger.Warning(errors.Wrap(err, "failed to close send channel"))
			}
			logger.Warning("disconnecting by killswitch")
			return ErrClientDisconnected
		case err := <-kill:
			logger.Warning(errors.Wrap(err, "send recv killed"))
			return ErrClientDisconnected
		}
	}
}

// Reset prepares the client for reconnection.
func (c *Client) Reset() {
	c.started = false
	c.session.Window = DefaultWindowSize
	c.lastWindowSize = DefaultWindowSize
}

// Finish checks the client has correctly received the sequence from the server
// and logs the result.
func (c *Client) Finish() error {
	if err := c.conn.Close(); err != nil {
		return errors.Wrap(err, "close client connection failed")
	}
	if !c.done {
		return ErrNotDone
	}
	if c.checksum == nil {
		return ErrMissingChecksum
	}
	sum, err := checksum.Sum(c.session.Sequence...)
	if err != nil {
		return errors.Wrap(err, "checksum failed")
	}
	if sum != *c.checksum {
		return ErrChecksumMismatch
	}
	logger.WithFields(logrus.Fields{
		"uuid":     c.uuid.String(),
		"sequence": c.session.Sequence,
		"checksum": *c.checksum,
	}).Info("client completed successfully")
	return nil
}
