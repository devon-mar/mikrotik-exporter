package collector

import (
	"fmt"
	"log/slog"

	"github.com/go-routeros/routeros/v3"
	"github.com/prometheus/client_golang/prometheus"
)

type collectorContext struct {
	ch     chan<- prometheus.Metric
	client *routeros.Client
	log    *slog.Logger
}

// assumes that the first sentence is the command
func (c *collectorContext) Run(sentences ...string) (*routeros.Reply, error) {
	reply, err := c.client.Run(sentences...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", sentences[0], err)
	}
	return reply, nil
}
