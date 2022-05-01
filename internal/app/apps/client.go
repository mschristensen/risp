package apps

import (
	"context"

	"risp/internal"
	"risp/internal/pkg/client"
	"risp/internal/pkg/validate"

	"github.com/pkg/errors"
)

// ClientAppCfg configures a ClientApp.
type ClientAppCfg interface {
	ApplyClientApp(*ClientApp) error
}

// ClientApp is the demo RISP client application.
type ClientApp struct {
	ServerAddr string `validate:"required"`
}

// NewClientApp creates a new ClientApp.
func NewClientApp(cfgs ...ClientAppCfg) (*ClientApp, error) {
	app := &ClientApp{}
	for _, cfg := range cfgs {
		if err := cfg.ApplyClientApp(app); err != nil {
			return nil, errors.Wrap(err, "apply ClientApp cfg failed")
		}
	}
	if err := validate.Validate().Struct(app); err != nil {
		return nil, errors.Wrap(err, "validate ClientApp failed")
	}
	return app, nil
}

func (app *ClientApp) Run(ctx context.Context, args []string) error {
	c, err := client.NewClient(
		client.WithRandomSequenceLength(),
		client.WithServerPort(uint16(internal.Port)),
	)
	if err != nil {
		return errors.Wrap(err, "create client failed")
	}
	if err := c.Run(ctx); err != nil {
		return errors.Wrap(err, "run client failed")
	}
	return errors.New("not implemented")
}
