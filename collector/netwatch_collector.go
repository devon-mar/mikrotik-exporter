package collector

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type netwatchCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newNetwatchCollector() routerOSCollector {
	c := &netwatchCollector{}
	c.init()
	return c
}

func (c *netwatchCollector) init() {
	c.props = []string{"host", "comment", "status"}
	labelNames := []string{"host", "comment"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[1:] {
		c.descriptions[p] = descriptionForPropertyName("netwatch", p, labelNames)
	}
}

func (c *netwatchCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *netwatchCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *netwatchCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/tool/netwatch/print", "?disabled=false", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		slog.Error(
			"error fetching netwatch metrics",
			"device", ctx.device.Name,
			"error", err,
		)
		return nil, err
	}

	return reply.Re, nil
}

func (c *netwatchCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	host := re.Map["host"]
	comment := re.Map["comment"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(p, host, comment, re, ctx)
	}
}

func (c *netwatchCollector) collectMetricForProperty(property, host, comment string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		var numericValue float64
		switch value {
		case "up":
			numericValue = 1
		case "unknown":
			numericValue = 0
		case "down":
			numericValue = -1
		default:
			slog.Error(
				"error parsing netwatch metric value",
				"device", ctx.device.Name,
				"host", host,
				"property", property,
				"value", value,
				"error", fmt.Errorf("unexpected netwatch status value"),
			)
		}
		ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, numericValue, host, comment)
	}
}
