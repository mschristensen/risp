package server

import (
	"context"
	"encoding/binary"
	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"
	"risp/internal/pkg/handler"
	"risp/internal/pkg/session"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var logger logrus.FieldLogger = logrus.StandardLogger()

type Server struct {
	session session.Store
}

// ServerCfg configures a Server.
type ServerCfg func(*Server) error

// WithSessionStore sets the session store for the server.
func WithSessionStore(store session.Store) ServerCfg {
	return func(s *Server) error {
		s.session = store
		return nil
	}
}

// NewServer creates a new Server with the given configuration.
func NewServer(cfgs ...ServerCfg) (*Server, error) {
	server := &Server{}
	for _, cfg := range cfgs {
		if err := cfg(server); err != nil {
			return nil, errors.Wrap(err, "apply Server cfg failed")
		}
	}
	return server, nil
}

func (s *Server) Connect(srv risppb.RISP_ConnectServer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// first, we expect to receive a client handshake with the clientUUID
	// and expected sequence length
	msg, err := srv.Recv()
	if err != nil {
		return errors.Wrap(err, "receive client handshake failed")
	}
	if msg.State != risppb.ConnectionState_CONNECTING {
		return errors.New("client handshake must be CONNECTING")
	}
	clientUUID, err := uuid.FromBytes(msg.Uuid)
	if err != nil {
		return errors.Wrap(err, "parse client UUID failed")
	}

	// load existing session state for client, or create new session state if none exists
	sess, err := s.session.Get(clientUUID)
	if err != nil && errors.Is(err, session.ErrSessionNotFound) {
		if err := s.session.New(clientUUID, uint16(msg.Len)); err != nil {
			return errors.Wrap(err, "new session failed")
		}
		sess, err = s.session.Get(clientUUID)
		if err != nil {
			return errors.Wrap(err, "get session after creating it failed")
		}
	}
	if len(sess.Sequence) != int(msg.Len) {
		return errors.New("sequence length mismatch")
	}
	// update the session state
	var ack, window uint16
	if len(msg.Ack) > 0 {
		ack = binary.BigEndian.Uint16(msg.Ack)
	}
	if len(msg.Window) > 0 {
		window = binary.BigEndian.Uint16(msg.Window)
	}
	if err = s.session.Set(clientUUID, ack, window); err != nil {
		return errors.Wrap(err, "set session failed")
	}

	in := make(chan *risppb.ClientMessage)
	out := make(chan *risppb.ServerMessage)
	hdl, err := handler.NewHandler(
		handler.WithClientUUID(clientUUID),
		handler.WithSessionStore(s.session),
	)
	if err != nil {
		return errors.Wrap(err, "create handler failed")
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := hdl.Run(ctx, in, out); err != nil {
			logger.Fatalln(err, "run worker failed")
		}
	}()
	go func() {
		defer close(in)
		defer wg.Done()
		for {
			msg, err := srv.Recv()
			if err != nil && status.Code(err) == codes.Canceled {
				logger.WithField("uuid", clientUUID).Warning("client disconnected")
				return
			}
			if err != nil {
				logger.Fatalln(err)
			}
			in <- msg
		}
	}()
	for msg := range out {
		if err := srv.Send(msg); err != nil {
			// TODO handle reconnection
			return errors.Wrap(err, "send message failed")
		}
	}
	wg.Wait()
	return nil
}
