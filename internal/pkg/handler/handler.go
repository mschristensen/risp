package handler

import (
	"context"
	"encoding/binary"
	"time"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"
	"risp/internal/pkg/checksum"
	"risp/internal/pkg/session"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type handler struct {
	clientUUID uuid.UUID
	session    session.Store
	offset     uint16
}

// HandlerCfg is configures a handler.
type HandlerCfg func(*handler) error

// WithSessionStore sets the session store.
func WithSessionStore(store session.Store) HandlerCfg {
	return func(w *handler) error {
		w.session = store
		return nil
	}
}

// WithClientUUID sets the client uuid.
func WithClientUUID(clientUUID uuid.UUID) HandlerCfg {
	return func(w *handler) error {
		w.clientUUID = clientUUID
		return nil
	}
}

// NewHandler creates a new handler.
func NewHandler(cfgs ...HandlerCfg) (*handler, error) {
	h := &handler{}
	for _, cfg := range cfgs {
		if err := cfg(h); err != nil {
			return nil, errors.Wrap(err, "apply handler cfg failed")
		}
	}
	return h, nil
}

// TODO only send messages up to window size

func (h *handler) handleMessage(ctx context.Context, msg *risppb.ClientMessage) (*risppb.ServerMessage, error) {
	switch msg.State {
	case risppb.ConnectionState_CONNECTING:
		// TODO: do not reinit sequence if already done, also do not allow len to be different
		if err := h.session.New(h.clientUUID, uint16(msg.Len)); err != nil {
			return nil, errors.Wrap(err, "new session failed")
		}
		return &risppb.ServerMessage{
			State:   risppb.ConnectionState_CONNECTED,
			Payload: 0, // TODO send first value in sequence
		}, nil
	case risppb.ConnectionState_CONNECTED:
		var ack, window uint16
		if len(msg.Ack) > 0 {
			ack = binary.BigEndian.Uint16(msg.Ack)
		}
		if len(msg.Window) > 0 {
			window = binary.BigEndian.Uint16(msg.Window)
		}
		if err := h.session.Set(h.clientUUID, ack, window); err != nil {
			return nil, errors.Wrap(err, "set session failed")
		}
		return nil, nil
		// update ack state
		// TODO handle window size, ack state, etc
		// w.return &risppb.ServerMessage{
		// 	State:   risppb.ConnectionState_CONNECTED,
		// 	Payload: payload,
		// }
	case risppb.ConnectionState_CLOSING:
		sess, err := h.session.Get(h.clientUUID)
		if err != nil {
			return nil, errors.Wrap(err, "get session failed")
		}
		return &risppb.ServerMessage{
			State:    risppb.ConnectionState_CLOSING,
			Checksum: checksum.Sum(sess.Sequence...),
		}, nil
	case risppb.ConnectionState_CLOSED:
		return &risppb.ServerMessage{
			State: risppb.ConnectionState_CLOSED,
		}, nil
	}
	return nil, errors.New("unhandled state")
}

// Run runs the handler.
func (h *handler) Run(ctx context.Context, in <-chan *risppb.ClientMessage, out chan<- *risppb.ServerMessage) error {
	defer close(out)
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-in:
			response, err := h.handleMessage(ctx, msg)
			if err != nil {
				return errors.Wrap(err, "handle message failed")
			}
			if response != nil {
				out <- response
			}
		case <-ticker.C:
			sess, err := h.session.Get(h.clientUUID)
			if err != nil {
				return errors.Wrap(err, "get session failed")
			}
			if int(h.offset) >= len(sess.Sequence) {
				continue
			}
			offset := make([]byte, 2)
			binary.BigEndian.PutUint16(offset, h.offset)
			out <- &risppb.ServerMessage{
				State:   risppb.ConnectionState_CONNECTED,
				Offset:  offset,
				Payload: sess.Sequence[h.offset],
			}
			h.offset++
			// todo pause at window size
		}
	}
}
