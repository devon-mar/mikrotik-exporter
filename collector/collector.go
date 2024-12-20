package collector

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	devices     []config.Device
	collectors  []routerOSCollector
	timeout     time.Duration
	enableTLS   bool
	insecureTLS bool
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
	client, err := routeros.DialContext(ctx, d.Address+":"+d.Port, d.User, d.Password)
	if err != nil {
		return nil, err
	}
	slog.Debug("got client", "device", d.Name)

	slog.Debug("trying to login", "device", d.Name)
	r, err := client.Run("/login", "=name="+d.User, "=password="+d.Password)
	if err != nil {
		return nil, err
	}
	ret, ok := r.Done.Map["ret"]
	if !ok {
		// Login method post-6.43 one stage, cleartext and no challenge
		if r.Done != nil {
			return client, nil
		}
		return nil, errors.New("RouterOS: /login: no ret (challenge) received")
	}

	// Login method pre-6.43 two stages, challenge
	b, err := hex.DecodeString(ret)
	if err != nil {
		return nil, fmt.Errorf("RouterOS: /login: invalid ret (challenge) hex string received: %s", err)
	}

	r, err = client.Run("/login", "=name="+d.User, "=response="+challengeResponse(b, d.Password))
	if err != nil {
		return nil, err
	}
	slog.Debug("done wth login", "device", d.Name)

	return client, nil

	//tlsCfg := &tls.Config{
	//	InsecureSkipVerify: c.insecureTLS,
	//}
	//	return routeros.DialTLSTimeout(d.Address+apiPortTLS, d.User, d.Password, tlsCfg, c.timeout)
}

func challengeResponse(cha []byte, password string) string {
	h := md5.New()
	h.Write([]byte{0})
	_, _ = io.WriteString(h, password)
	h.Write(cha)
	return fmt.Sprintf("00%x", h.Sum(nil))
}
