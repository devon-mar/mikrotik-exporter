package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type poolCollector struct {
	usedCountDesc *prometheus.Desc
}

func (c *poolCollector) init() {
	const prefix = "ip_pool"

	labelNames := []string{"ip_version", "pool"}
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
	return c.collectForIPVersion(ctx, "4", "ip")
}

func (c *poolCollector) collectForIPVersion(ctx *collectorContext, ipVersion, topic string) error {
	names, err := c.fetchPoolNames(ctx, ipVersion, topic)
	if err != nil {
		return err
	}

	for _, n := range names {
		err := c.collectForPool(ctx, ipVersion, topic, n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *poolCollector) fetchPoolNames(ctx *collectorContext, ipVersion, topic string) ([]string, error) {
	reply, err := ctx.Run(fmt.Sprintf("/%s/pool/print", topic), "=.proplist=name")
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *poolCollector) collectForPool(ctx *collectorContext, ipVersion, topic, pool string) error {
	reply, err := ctx.client.Run(fmt.Sprintf("/%s/pool/used/print", topic), fmt.Sprintf("?pool=%s", pool), "=count-only=")
	if err != nil {
		ctx.log.Error(
			"error fetching pool counts",
			"pool", pool,
			"ip_version", ipVersion,
			"err", err,
		)
		return err
	}
	if reply.Done.Map["ret"] == "" {
		return nil
	}
	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		ctx.log.Error(
			"error parsing pool counts",
			"pool", pool,
			"ip_version", ipVersion,
			"err", err,
		)
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(c.usedCountDesc, prometheus.GaugeValue, v, ipVersion, pool)
	return nil
}
