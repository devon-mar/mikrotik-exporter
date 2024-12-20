package collector

import (
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type bgpCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newBGPCollector() routerOSCollector {
	c := &bgpCollector{}
	c.init()
	return c
}

func (c *bgpCollector) init() {
	c.props = []string{"name", "remote-as", "state", "prefix-count", "updates-sent", "updates-received", "withdrawn-sent", "withdrawn-received"}

	const prefix = "bgp"
	labelNames := []string{"session", "asn"}

	c.descriptions = make(map[string]*prometheus.Desc)
	c.descriptions["state"] = description(prefix, "up", "BGP session is established (up = 1)", labelNames)

	for _, p := range c.props[3:] {
		c.descriptions[p] = descriptionForPropertyName(prefix, p, labelNames)
	}
}

func (c *bgpCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *bgpCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(ctx, re)
	}

	return nil
}

func (c *bgpCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.Run("/routing/bgp/peer/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		return nil, err
	}

	return reply.Re, nil
}

func (c *bgpCollector) collectForStat(ctx *collectorContext, re *proto.Sentence) {
	asn := re.Map["remote-as"]
	session := re.Map["name"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(ctx, p, session, asn, re)
	}
}

func (c *bgpCollector) collectMetricForProperty(ctx *collectorContext, property, session, asn string, re *proto.Sentence) {
	desc := c.descriptions[property]
	v, err := c.parseValueForProperty(property, re.Map[property])
	if err != nil {
		ctx.log.Error(
			"error parsing bgp metric value",
			"session", session,
			"property", property,
			"value", re.Map[property],
			"err", err,
		)
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, session, asn)
}

func (c *bgpCollector) parseValueForProperty(property, value string) (float64, error) {
	if property == "state" {
		if value == "established" {
			return 1, nil
		}

		return 0, nil
	}

	if value == "" {
		return 0, nil
	}

	return strconv.ParseFloat(value, 64)
}
