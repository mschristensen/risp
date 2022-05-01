package client

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"
	"risp/internal/pkg/checksum"
	"risp/internal/pkg/log"
	"risp/internal/pkg/session"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var logger logrus.FieldLogger = logrus.StandardLogger()

// TODO handle window size that is not a factor of sequence length

// DefaultWindowSize is the default window size for the client.
const DefaultWindowSize = 3

// Client implements the client behaviour of RISP.
type Client struct {
	serverAddr string
	uuid       uuid.UUID
	session    session.Session

	started  bool
	closing  bool
	done     bool
	checksum *uint64

	conn    *grpc.ClientConn
	channel risppb.RISP_ConnectClient
}

// ClientCfg configures a Client.
type ClientCfg func(*Client) error

// WithServerPort sets the server port to connect to.
func WithServerPort(p uint16) ClientCfg {
	return func(c *Client) error {
		c.serverAddr = fmt.Sprintf("localhost:%d", p)
		return nil
	}
}

// WithSequenceLength sets the length of the sequence.
func WithSequenceLength(l uint16) ClientCfg {
	return func(c *Client) error {
		c.session.Sequence = make([]*uint32, l)
		return nil
	}
}

// WithRandomSequenceLength sets the sequence length to a random non-zero value in the supported range.
func WithRandomSequenceLength() ClientCfg {
	return func(c *Client) error {
		c.session.Sequence = make([]*uint32, rand.Intn(math.MaxUint16)+1)
		return nil
	}
}

// NewClient creates a new Client with the given configuration.
func NewClient(cfgs ...ClientCfg) (*Client, error) {
	client := &Client{}
	for _, cfg := range cfgs {
		if err := cfg(client); err != nil {
			return nil, errors.Wrap(err, "apply Client cfg failed")
		}
	}
	client.uuid = uuid.New()
	client.session.Window = DefaultWindowSize
	return client, nil
}

// Connect establishes the connection to the server.
func (c *Client) Connect(ctx context.Context) error {
	if c.conn != nil {
		// TODO ensure conn gets closed even if there is never a redial
		if err := c.conn.Close(); err != nil {
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
	c.channel, err = risppb.NewRISPClient(c.conn).Connect(ctx)
	if err != nil {
		return errors.Wrap(err, "call connect failed")
	}
	return nil
}

// sendRecv sends messages to the server that are received on the inbound channel,
// and receives messages from the server and sends them on the returned on the outbound channel.
func (c *Client) sendRecv(ctx context.Context, in chan *risppb.ClientMessage) (chan *risppb.ServerMessage, error) {
	out := make(chan *risppb.ServerMessage)
	go func() {
		defer close(out)
		for {
			msg, err := c.channel.Recv()
			if err != nil && status.Code(err) == codes.Canceled {
				if c.done {
					return
				}
				// TODO handle reconnection
				panic(err)
			}
			out <- msg
		}
	}()
	go func() {
		for msg := range in {
			if err := c.channel.Send(msg); err != nil {
				// TODO handle error
				panic(err)
			}
		}
	}()
	return out, nil
}

// handleMessage updates the client state using the message from the server.
func (c *Client) handleMessage(ctx context.Context, msg *risppb.ServerMessage) error {
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
func (c *Client) nextMessage() (*risppb.ClientMessage, error) {
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
		return msg, nil
	}
	if c.closing && c.checksum != nil {
		msg.State = risppb.ConnectionState_CLOSED
		return msg, nil
	}
	if c.checksum == nil && c.session.Ack == uint16(len(c.session.Sequence)) {
		msg.State = risppb.ConnectionState_CLOSING
		return msg, nil
	}
	return msg, nil
}

func (c *Client) sendNextMessage(out chan *risppb.ClientMessage) error {
	msg, err := c.nextMessage()
	if err != nil {
		return errors.Wrap(err, "next message failed")
	}
	out <- msg
	logger.WithFields(log.ClientMessageToFields(msg)).Info("sent message")
	return nil
}

func (c *Client) Finish() error {
	if !c.done {
		return errors.New("client not finished")
	}
	if c.checksum == nil {
		return errors.New("checksum not received")
	}
	sum, err := checksum.Sum(c.session.Sequence...)
	if err != nil {
		return errors.Wrap(err, "checksum failed")
	}
	if sum != *c.checksum {
		return errors.New("checksum mismatch")
	}
	logger.WithFields(logrus.Fields{
		"uuid":     c.uuid.String(),
		"sequence": c.session.Sequence,
		"checksum": c.checksum,
	}).Info("client completed successfully")
	return nil
}

// Run runs client-side RISP protocol to receive the integer stream from the server.
func (c *Client) Run(ctx context.Context) error {
	out := make(chan *risppb.ClientMessage)
	defer close(out)
	in, err := c.sendRecv(ctx, out)
	if err != nil {
		return errors.Wrap(err, "connect failed")
	}
	msg, err := c.nextMessage()
	if err != nil {
		return errors.Wrap(err, "next message failed")
	}
	out <- msg
	logger.WithFields(log.ClientMessageToFields(msg)).Info("sent message")
	ticker := time.NewTicker(2 * time.Second)
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
				if err := c.Finish(); err != nil {
					return errors.Wrap(err, "finish failed")
				}
				return nil
			}
		case <-ticker.C:
			if c.session.Window == 0 || c.session.Ack == uint16(len(c.session.Sequence)) {
				c.session.Window = DefaultWindowSize // TODO dynamically size
				if err := c.sendNextMessage(out); err != nil {
					return errors.Wrap(err, "send next message failed")
				}
			}
		}
	}
}
