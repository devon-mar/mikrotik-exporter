package collector

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace  = "mikrotik"
	apiPort    = "8728"
	apiPortTLS = "8729"
	dnsPort    = 53

	// DefaultTimeout defines the default timeout when connecting to a router
	DefaultTimeout = 5 * time.Second
)

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"mikrotik_exporter: duration of a device collector scrape",
		[]string{},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"mikrotik_exporter: whether a device collector succeeded",
		[]string{},
		nil,
	)
)

type collector struct {
	devices    []device
	collectors []routerOSCollector
	// if nil, tls will not be used to connect to the device
	tlsCfg *tls.Config
}

func (c *collector) collectForDevice(ctx context.Context, d device, ch chan<- prometheus.Metric) {
	begin := time.Now()

	err := c.connectAndCollect(ctx, &d, ch)

	duration := time.Since(begin)
	var success float64
	if err != nil {
		slog.Error("collector failed", "target", d.Address, "duration", duration.Seconds(), "err", err)
		success = 0
	} else {
		slog.Debug("collector succeeded", "target", d.Address, "duration", duration.Seconds())
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds())
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success)
}

func (c *collector) connectAndCollect(ctx context.Context, d *device, ch chan<- prometheus.Metric) error {
	logger := slog.With("target", d.Address)

	cl, err := c.connect(ctx, d)
	if err != nil {
		logger.Error(
			"error dialing device",
			"error", err,
		)
		return err
	}
	defer cl.Close()

	for _, co := range c.collectors {
		ctx := &collectorContext{
			ch:     ch,
			device: d,
			client: cl,
			log:    logger,
		}
		err = co.collect(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *collector) connect(ctx context.Context, d *device) (*routeros.Client, error) {
	var client *routeros.Client
	var err error
	if c.tlsCfg != nil {
		client, err = routeros.DialTLSContext(ctx, d.Address, d.User, d.Password, c.tlsCfg)
	} else {
		client, err = routeros.DialContext(ctx, d.Address, d.User, d.Password)
	}
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return client, nil
}
