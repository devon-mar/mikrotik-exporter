package collector

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"mikrotik-exporter/config"

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
	devices    []config.Device
	collectors []routerOSCollector
	// if nil, tls will not be used to connect to the device
	tlsCfg *tls.Config
}

func (c *collector) getIdentity(ctx context.Context, d *config.Device) error {
	cl, err := c.connect(ctx, d)
	if err != nil {
		slog.Error(
			"error dialing device fetching identity",
			"device", d.Name,
			"error", err,
		)
		return err
	}
	defer cl.Close()
	reply, err := cl.Run("/system/identity/print")
	if err != nil {
		slog.Error(
			"error fetching ethernet interfaces",
			"device", d.Name,
			"error", err,
		)
		return err
	}
	for _, id := range reply.Re {
		d.Name = id.Map["name"]
	}
	return nil
}

func (c *collector) collectForDevice(ctx context.Context, d config.Device, ch chan<- prometheus.Metric) {
	begin := time.Now()

	err := c.connectAndCollect(ctx, &d, ch)

	duration := time.Since(begin)
	var success float64
	if err != nil {
		slog.Error("collector failed", "collector", d.Name, "duration", duration.Seconds(), "err", err)
		success = 0
	} else {
		slog.Debug("collector succeeded", "collector", d.Name, "duration", duration.Seconds())
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds())
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success)
}

func (c *collector) connectAndCollect(ctx context.Context, d *config.Device, ch chan<- prometheus.Metric) error {
	cl, err := c.connect(ctx, d)
	if err != nil {
		slog.Error(
			"error dialing device",
			"device", d.Name,
			"error", err,
		)
		return err
	}
	defer cl.Close()

	for _, co := range c.collectors {
		ctx := &collectorContext{ch, d, cl}
		err = co.collect(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *collector) connect(ctx context.Context, d *config.Device) (*routeros.Client, error) {
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
