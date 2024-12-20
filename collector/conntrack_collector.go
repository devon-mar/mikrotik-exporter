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
	reply, err := ctx.Run("/ip/firewall/connection/tracking/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		return err
	}

	for _, re := range reply.Re {
		c.collectMetricForProperty(ctx, "total-entries", c.totalEntriesDesc, re)
		c.collectMetricForProperty(ctx, "max-entries", c.maxEntriesDesc, re)
	}

	return nil
}

func (c *conntrackCollector) collectMetricForProperty(ctx *collectorContext, property string, desc *prometheus.Desc, re *proto.Sentence) {
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
