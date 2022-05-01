package apps

import (
	"context"
	"strconv"

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
	Port uint16 `validate:"required"`
}

// NewClientApp creates a new ClientApp.
func NewClientApp(cfgs ...ClientAppCfg) (*ClientApp, error) {
	app := &ClientApp{}
	for _, cfg := range cfgs {
		if err := cfg.ApplyClientApp(app); err != nil {
			return nil, errors.Wrap(err, "apply ClientApp cfg failed")
		}
	}
	if app.Port == 0 {
		app.Port = uint16(internal.Port)
	}
	if err := validate.Validate().Struct(app); err != nil {
		return nil, errors.Wrap(err, "validate ClientApp failed")
	}
	return app, nil
}

// Run runs the demo RISP client application.
func (app *ClientApp) Run(ctx context.Context, args []string) error {
	cfgs := []client.Cfg{
		client.WithServerPort(app.Port),
	}
	if len(args) > 1 {
		sequenceLength, err := strconv.ParseUint(args[1], 10, 16)
		if err != nil {
			return errors.Wrap(err, "parse sequence length argument failed")
		}
		cfgs = append(cfgs, client.WithSequenceLength(uint16(sequenceLength)))
	} else {
		cfgs = append(cfgs, client.WithRandomSequenceLength())
	}
	c, err := client.NewClient(cfgs...)
	if err != nil {
		return errors.Wrap(err, "create client failed")
	}
	if err := c.Connect(ctx); err != nil {
		return errors.Wrap(err, "connect client failed")
	}
	if err := c.Run(ctx); err != nil {
		return errors.Wrap(err, "run client failed")
	}
	return nil
}
