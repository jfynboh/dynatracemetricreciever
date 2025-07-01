package dynatracemetricreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// func NewFactory() receiver.Factory {
// 	return receiver.NewFactory(
// 		typeStr,
// 		createDefaultConfig,
// 		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelDevelopment),
// 	)
// }

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType("dynatracemetric"),
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelDevelopment))
}

func createDefaultConfig() component.Config {
	return &Config{
		Endpoint: "localhost:14499",
	}
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	rCfg := cfg.(*Config)
	return newDynatraceMetricReceiver(params, *rCfg, consumer)
}
