package client

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var logger logrus.FieldLogger = logrus.StandardLogger()

// DefaultWindowSize is the default window size for the client.
const DefaultWindowSize = 1

// Client implements the client behaviour of RISP.
type Client struct {
	serverAddr     string
	uuid           uuid.UUID
	window         uint16 // TODO handle window size that is not a factor of sequence length
	ack            uint16
	sequenceLength uint16
	sequence       []uint32
	checksum       []byte
	started        bool
	done           bool

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
		c.sequenceLength = l
		return nil
	}
}

// WithRandomSequenceLength sets the sequence length to a random non-zero value in the supported range.
func WithRandomSequenceLength() ClientCfg {
	return func(c *Client) error {
		c.sequenceLength = uint16(rand.Intn(math.MaxUint16) + 1)
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
	client.sequence = make([]uint32, client.sequenceLength)
	client.uuid = uuid.New()
	client.window = DefaultWindowSize
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
		// TODO: use TLS
		grpc.WithTransportCredentials(insecure.NewCredentials()),
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
			if err != nil {
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

// handleMessage handles a message from the server.
func (c *Client) handleMessage(ctx context.Context, msg *risppb.ServerMessage) error {
	if msg.State == risppb.ConnectionState_CLOSING {
		if c.ack != 1<<c.sequenceLength-1 {
			return errors.New("received closing message before all items received")
		}
		c.checksum = msg.Checksum
		c.window = 0
		return nil
	}
	if msg.State == risppb.ConnectionState_CLOSED {
		if c.ack != 1<<c.sequenceLength-1 {
			return errors.New("received closed message before all items received")
		}
		if len(c.checksum) == 0 {
			return errors.New("received closed message before checksum received")
		}
		c.window = 0
		c.done = true
		return nil
	}
	var sequenceIndex uint16
	if len(msg.Offset) > 0 {
		sequenceIndex = binary.BigEndian.Uint16(msg.Offset)
		// update record of what we have seen
		c.ack |= (1 << sequenceIndex) // TODO: use a more efficient encoding scheme
	}
	// store the item at the correct place in the sequence, as described by the offset
	c.sequence[sequenceIndex] = msg.Payload
	// reduce the window size
	c.window--
	return nil
}

// nextMessage prepares the next message to send to the server based on the current client state.
func (c *Client) nextMessage() (*risppb.ClientMessage, error) {
	msg := &risppb.ClientMessage{
		Uuid: c.uuid[:],
		Len:  uint32(c.sequenceLength),
	}

	if !c.started {
		msg.State = risppb.ConnectionState_CONNECTING
		c.started = true
		return msg, nil
	}
	if len(c.checksum) > 0 {
		msg.State = risppb.ConnectionState_CLOSED
		return msg, nil
	}
	if c.ack == 1<<c.sequenceLength-1 {
		msg.State = risppb.ConnectionState_CLOSING
		return msg, nil
	}
	msg.State = risppb.ConnectionState_CONNECTED

	bs, err := c.uuid.MarshalBinary()
	if err != nil {
		return nil, errors.Wrap(err, "uuid marshal failed")
	}
	msg.Uuid = bs

	window := make([]byte, 2)
	binary.BigEndian.PutUint16(window, c.window)
	msg.Window = window

	ack := make([]byte, 2)
	binary.BigEndian.PutUint16(ack, c.ack)
	msg.Ack = ack

	return msg, nil
}

// Run runs client-side RISP algorithm to receive the integer stream from the server.
func (c *Client) Run(ctx context.Context) error {
	outbox := make(chan *risppb.ClientMessage)
	defer close(outbox)
	inbox, err := c.sendRecv(ctx, outbox)
	if err != nil {
		return errors.Wrap(err, "connect failed")
	}
	msg, err := c.nextMessage()
	if err != nil {
		return errors.Wrap(err, "next message failed")
	}
	outbox <- msg
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-inbox:
			logger.WithFields(logrus.Fields{
				"state":    msg.State.String(),
				"offset":   msg.Offset,
				"payload":  msg.Payload,
				"checksum": msg.Checksum,
			}).Info("received message")
			if err := c.handleMessage(ctx, msg); err != nil {
				return errors.Wrap(err, "handle message failed")
			}
			if c.done {
				return nil
			}
			if c.window == 0 { // OR timeout occurs!
				c.window = DefaultWindowSize // TODO dynamically size
				msg, err := c.nextMessage()
				if err != nil {
					return errors.Wrap(err, "next message failed")
				}
				outbox <- msg
				logger.WithFields(logrus.Fields{
					"state":  msg.State.String(),
					"uuid":   msg.Uuid,
					"ack":    msg.Ack,
					"len":    msg.Len,
					"window": msg.Window,
				}).Info("sent message")
			}
		}
	}
}
