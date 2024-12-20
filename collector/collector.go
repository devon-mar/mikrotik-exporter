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
	collectors []routerOSCollector
	// if nil, tls will not be used to connect to the device
	tlsCfg *tls.Config

	usernameStr string
	passwordStr string
}

func (c *collector) credentials() (string, string, error) {
	return c.usernameStr, c.passwordStr, nil
}

func (c *collector) collectForDevice(ctx context.Context, target string, ch chan<- prometheus.Metric) {
	begin := time.Now()

	err := c.connectAndCollect(ctx, target, ch)

	duration := time.Since(begin)
	var success float64
	if err != nil {
		slog.Error("collector failed", "target", target, "duration", duration.Seconds(), "err", err)
		success = 0
	} else {
		slog.Debug("collector succeeded", "target", target, "duration", duration.Seconds())
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds())
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success)
}

func (c *collector) connectAndCollect(ctx context.Context, target string, ch chan<- prometheus.Metric) error {
	logger := slog.With("target", target)

	cl, err := c.connect(ctx, target)
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

func (c *collector) connect(ctx context.Context, target string) (*routeros.Client, error) {
	var client *routeros.Client
	var err error
	username, password, err := c.credentials()
	if err != nil {
		return nil, fmt.Errorf("credentials: %w", err)
	}

	if c.tlsCfg != nil {
		client, err = routeros.DialTLSContext(ctx, target, username, password, c.tlsCfg)
	} else {
		client, err = routeros.DialContext(ctx, target, username, password)
	}
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return client, nil
}
