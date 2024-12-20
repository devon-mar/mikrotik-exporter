package collector

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type poolCollector struct {
	usedCountDesc *prometheus.Desc
}

func (c *poolCollector) init() {
	const prefix = "ip_pool"

	labelNames := []string{"name", "address", "ip_version", "pool"}
	c.usedCountDesc = description(prefix, "pool_used_count", "number of used IP/prefixes in a pool", labelNames)
}

func newPoolCollector() routerOSCollector {
	c := &poolCollector{}
	c.init()
	return c
}

func (c *poolCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.usedCountDesc
}

func (c *poolCollector) collect(ctx *collectorContext) error {
	return c.collectForIPVersion("4", "ip", ctx)
}

func (c *poolCollector) collectForIPVersion(ipVersion, topic string, ctx *collectorContext) error {
	names, err := c.fetchPoolNames(ipVersion, topic, ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		err := c.collectForPool(ipVersion, topic, n, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *poolCollector) fetchPoolNames(ipVersion, topic string, ctx *collectorContext) ([]string, error) {
	reply, err := ctx.client.Run(fmt.Sprintf("/%s/pool/print", topic), "=.proplist=name")
	if err != nil {
		slog.Error(
			"error fetching pool names",
			"device", ctx.device.Name,
			"error", err,
		)
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *poolCollector) collectForPool(ipVersion, topic, pool string, ctx *collectorContext) error {
	reply, err := ctx.client.Run(fmt.Sprintf("/%s/pool/used/print", topic), fmt.Sprintf("?pool=%s", pool), "=count-only=")
	if err != nil {
		slog.Error(
			"error fetching pool counts",
			"pool", pool,
			"ip_version", ipVersion,
			"device", ctx.device.Name,
			"error", err,
		)
		return err
	}
	if reply.Done.Map["ret"] == "" {
		return nil
	}
	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		slog.Error(
			"error parsing pool counts",
			"pool", pool,
			"ip_version", ipVersion,
			"device", ctx.device.Name,
			"error", err,
		)
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(c.usedCountDesc, prometheus.GaugeValue, v, ctx.device.Name, ctx.device.Address, ipVersion, pool)
	return nil
}
