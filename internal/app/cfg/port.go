// Package cfg implements functionaltiy to configure an app.
//
// The configuration objects defined here need only be implemented once,
// but can be applied to multiple types.
//
// In order to add support for a new type, the configuration
// need only implement an ApplyX method.
//
package cfg

import (
	"risp/internal"
	"risp/internal/app/apps"
)

// PortCfg is configuration for the RISP server port.
type PortCfg struct {
	port uint16
}

// NewPortCfg creates a new PortCfg from the given config.
func NewPortCfg(port uint16) *PortCfg {
	return &PortCfg{
		port: port,
	}
}

// PortFromEnv creates a new PortCfg from the current environment.
func PortFromEnv() *PortCfg {
	return &PortCfg{
		port: uint16(internal.Port),
	}
}

// ApplyClientApp applies the PortCfg to a ClientApp.
func (cfg PortCfg) ApplyClientApp(app *apps.ClientApp) error {
	app.Port = cfg.port
	return nil
}

// ApplyServerApp applies the PortCfg to a ServerApp.
func (cfg PortCfg) ApplyServerApp(app *apps.ServerApp) error {
	app.Port = cfg.port
	return nil
}
