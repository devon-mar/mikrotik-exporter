package collector

import (
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type conntrackCollector struct {
	props            []string
	totalEntriesDesc *prometheus.Desc
	maxEntriesDesc   *prometheus.Desc
}

func newConntrackCollector() routerOSCollector {
	const prefix = "conntrack"

	labelNames := []string{}
	return &conntrackCollector{
		props:            []string{"total-entries", "max-entries"},
		totalEntriesDesc: description(prefix, "entries", "Number of tracked connections", labelNames),
		maxEntriesDesc:   description(prefix, "max_entries", "Conntrack table capacity", labelNames),
	}
}

func (c *conntrackCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalEntriesDesc
	ch <- c.maxEntriesDesc
}

func (c *conntrackCollector) collect(ctx *collectorContext) error {
	reply, err := ctx.client.Run("/ip/firewall/connection/tracking/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		ctx.log.Error(
			"error fetching conntrack table metrics",
			"err", err,
		)
		return err
	}

	for _, re := range reply.Re {
		c.collectMetricForProperty("total-entries", c.totalEntriesDesc, re, ctx)
		c.collectMetricForProperty("max-entries", c.maxEntriesDesc, re, ctx)
	}

	return nil
}

func (c *conntrackCollector) collectMetricForProperty(property string, desc *prometheus.Desc, re *proto.Sentence, ctx *collectorContext) {
	if re.Map[property] == "" {
		return
	}
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		ctx.log.Error(
			"error parsing conntrack metric value",
			"property", property,
			"value", re.Map[property],
			"err", err,
		)
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v)
}
