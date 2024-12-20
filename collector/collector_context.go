package collector

import (
	"log/slog"

	"github.com/go-routeros/routeros/v3"
	"github.com/prometheus/client_golang/prometheus"
)

type collectorContext struct {
	ch     chan<- prometheus.Metric
	device *device
	client *routeros.Client
	log    *slog.Logger
}
