package dynatracemetricreceiver

import "go.opentelemetry.io/collector/component"

// Config defines configuration for the dynatrace receiver
type Config struct {
	component.Config `mapstructure:",squash"` // marker interface, no methods

	Endpoint string `mapstructure:"endpoint"`
}
