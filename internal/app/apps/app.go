// Package apps implements the entrypoints to the application.
//
// Multiple apps can be implemented here.
// Each can be configured independently, but the configuration objects can be reused.
package apps

import (
	"context"
)

// AppCfg configures an App.
type AppCfg interface {
	ClientAppCfg
	ServerAppCfg
	// ... add more here to configure additional apps
}

// App runs an app.
type App interface {
	Run(ctx context.Context, args []string) error
}
