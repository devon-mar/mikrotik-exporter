package collector

import (
	"strconv"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type healthCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newhealthCollector() routerOSCollector {
	c := &healthCollector{}
	c.init()
	return c
}

func (c *healthCollector) init() {
	c.props = []string{"voltage", "temperature", "cpu-temperature"}

	labelNames := []string{}
	helpText := []string{"Input voltage to the RouterOS board, in volts", "Temperature of RouterOS board, in degrees Celsius", "Temperature of RouterOS CPU, in degrees Celsius"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for i, p := range c.props {
		c.descriptions[p] = descriptionForPropertyNameHelpText("health", p, labelNames, helpText[i])
	}
}

func (c *healthCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *healthCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		if metric, ok := re.Map["name"]; ok {
			c.collectMetricForProperty(metric, re, ctx)
		} else {
			c.collectForStat(re, ctx)
		}
	}

	return nil
}

func (c *healthCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.Run("/system/health/print")
	if err != nil {
		return nil, err
	}

	return reply.Re, nil
}

func (c *healthCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	for _, p := range c.props[:3] {
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *healthCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *collectorContext) {
	var v float64
	var err error

	name := property
	value := re.Map[property]

	if value == "" {
		var ok bool
		if value, ok = re.Map["value"]; !ok {
			return
		}
	}
	v, err = strconv.ParseFloat(value, 64)
	if err != nil {
		ctx.log.Error(
			"error parsing system health metric value",
			"property", name,
			"value", value,
			"err", err,
		)
		return
	}

	desc := c.descriptions[name]
	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v)
}
