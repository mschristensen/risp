package server

import (
	"context"
	"time"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"
	"risp/internal/pkg/checksum"
	"risp/internal/pkg/log"
	"risp/internal/pkg/session"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var logger logrus.FieldLogger = logrus.StandardLogger()

// Handler implements RISP for a specific client.
type Handler struct {
	clientUUID uuid.UUID
	store      session.Store
	session    session.Session // current session state

	closing bool
	done    bool
}

// NewHandler creates a new handler.
func NewHandler(clientUUID uuid.UUID, store session.Store) *Handler {
	return &Handler{
		clientUUID: clientUUID,
		store:      store,
	}
}

// handleMessage updates the server state based on the client message.
func (h *Handler) handleMessage(msg *risppb.ClientMessage) error {
	switch msg.State {
	case risppb.ConnectionState_CONNECTING, risppb.ConnectionState_CONNECTED:
		// update session state according to the client message
		h.session.Ack = uint16(msg.Ack)
		h.session.Window = uint16(msg.Window)
		if err := h.store.Set(h.clientUUID, h.session); err != nil {
			return errors.Wrap(err, "set session failed")
		}
		// TODO: do not reinit sequence if already done, also do not allow len to be different
		return nil
	case risppb.ConnectionState_CLOSING:
		h.closing = true
		return nil
	case risppb.ConnectionState_CLOSED:
		if h.done {
			return nil
		}
		if err := h.store.Clear(h.clientUUID); err != nil {
			return errors.Wrap(err, "clear session failed")
		}
		h.done = true
		return nil
	}
	return errors.New("unhandled state")
}

// nextMessage prepares the next message to send to the client based on the current handler state.
func (h *Handler) nextMessage() (*risppb.ServerMessage, error) {
	msg := &risppb.ServerMessage{
		State: risppb.ConnectionState_CONNECTED,
	}
	if h.done {
		msg.State = risppb.ConnectionState_CLOSED
		return msg, nil
	}
	if h.closing {
		msg.State = risppb.ConnectionState_CLOSING
		sum, err := checksum.Sum(h.session.Sequence...)
		if err != nil {
			return nil, errors.Wrap(err, "checksum failed")
		}
		msg.Checksum = sum
		return msg, nil
	}

	// stop sending messages if we have sent all the messages
	// or if we have exhausted the window size
	if h.session.Ack == uint16(len(h.session.Sequence)) || h.session.Window == 0 {
		return nil, nil
	}

	msg.Index = uint32(h.session.Ack)
	msg.Payload = *h.session.Sequence[h.session.Ack]

	return msg, nil
}

// Run runs the handler.
func (h *Handler) Run(ctx context.Context, in <-chan *risppb.ClientMessage, out chan<- *risppb.ServerMessage) error {
	defer close(out)
	ticker := time.NewTicker(time.Second)

	// initialise the handler state with the stored client session state
	sess, err := h.store.Get(h.clientUUID)
	if err != nil {
		return errors.Wrap(err, "get session failed")
	}
	h.session = sess

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-in:
			if !ok || msg == nil {
				return nil
			}
			logger.WithFields(log.ClientMessageToFields(msg)).Info("received message")
			if err := h.handleMessage(msg); err != nil {
				return errors.Wrap(err, "handle message failed")
			}
		case <-ticker.C:
			msg, err := h.nextMessage()
			if err != nil {
				return errors.Wrap(err, "next message failed")
			}
			if msg != nil {
				out <- msg
				logger.WithFields(log.ServerMessageToFields(msg)).Info("sent message")
				h.session.Window--
				h.session.Ack++
			}
		}
	}
}
