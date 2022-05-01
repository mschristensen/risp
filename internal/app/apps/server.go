package apps

import (
	"context"
	"fmt"
	"net"

	"risp/internal"
	"risp/internal/pkg/server"
	"risp/internal/pkg/session"
	"risp/internal/pkg/validate"

	risppb "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var logger *logrus.Logger = logrus.StandardLogger()

// ServerAppCfg configures a ServerApp.
type ServerAppCfg interface {
	ApplyServerApp(*ServerApp) error
}

// ServerApp is the demo RISP client application.
type ServerApp struct {
	Port uint16 `validate:"required"`
}

// NewServerApp creates a new ServerApp.
func NewServerApp(cfgs ...ServerAppCfg) (*ServerApp, error) {
	app := &ServerApp{}
	for _, cfg := range cfgs {
		if err := cfg.ApplyServerApp(app); err != nil {
			return nil, errors.Wrap(err, "apply ServerApp cfg failed")
		}
	}
	if app.Port == 0 {
		app.Port = uint16(internal.Port)
	}
	if err := validate.Validate().Struct(app); err != nil {
		return nil, errors.Wrap(err, "validate ServerApp failed")
	}
	return app, nil
}

func (app *ServerApp) Run(ctx context.Context, args []string) error {
	server, err := server.NewServer(
		server.WithSessionStore(session.NewMemoryStore()),
	)
	if err != nil {
		return errors.Wrap(err, "new server failed")
	}
	grpcServer := grpc.NewServer()
	risppb.RegisterRISPServer(grpcServer, server)

	// stop the server when the context is done
	go func() {
		<-ctx.Done()
		grpcServer.Stop()
	}()

	logger.WithField("port", app.Port).Info("gRPC server listening")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", app.Port))
	if err != nil {
		return errors.Wrap(err, "listen failed")
	}
	if err := grpcServer.Serve(lis); err != nil {
		return errors.Wrap(err, "server failed")
	}
	return nil
}
