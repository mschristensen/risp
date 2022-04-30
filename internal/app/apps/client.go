package apps

import (
	"context"

	"risp/internal/pkg/validate"

	"github.com/pkg/errors"
)

// ClientAppCfg configures a ClientApp.
type ClientAppCfg interface {
	ApplyClientApp(*ClientApp) error
}

// ClientApp is the demo RISP client application.
type ClientApp struct {
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
	return errors.New("not implemented")
}
