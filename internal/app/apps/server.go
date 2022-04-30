package apps

import (
	"context"

	"risp/internal/pkg/validate"

	"github.com/pkg/errors"
)

// ServerAppCfg configures a ServerApp.
type ServerAppCfg interface {
	ApplyServerApp(*ServerApp) error
}

// ServerApp is the demo RISP client application.
type ServerApp struct {
}

// NewServerApp creates a new ServerApp.
func NewServerApp(cfgs ...ServerAppCfg) (*ServerApp, error) {
	app := &ServerApp{}
	for _, cfg := range cfgs {
		if err := cfg.ApplyServerApp(app); err != nil {
			return nil, errors.Wrap(err, "apply ServerApp cfg failed")
		}
	}
	if err := validate.Validate().Struct(app); err != nil {
		return nil, errors.Wrap(err, "validate ServerApp failed")
	}
	return app, nil
}

func (app *ServerApp) Run(ctx context.Context, args []string) error {
	return errors.New("not implemented")
}
