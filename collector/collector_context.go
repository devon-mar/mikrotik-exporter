package collector

import (
	"log/slog"

	"mikrotik-exporter/config"

	"github.com/go-routeros/routeros/v3"
	"github.com/prometheus/client_golang/prometheus"
)

type collectorContext struct {
	ch     chan<- prometheus.Metric
	device *config.Device
	client *routeros.Client
	log    *slog.Logger
}
