package collector

import (
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type ipsecCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newIpsecCollector() routerOSCollector {
	c := &ipsecCollector{}
	c.init()
	return c
}

func (c *ipsecCollector) init() {
	c.props = []string{"src-address", "dst-address", "ph2-state", "invalid", "active", "comment"}

	labelNames := []string{"srcdst", "comment"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[1:] {
		c.descriptions[p] = descriptionForPropertyName("ipsec", p, labelNames)
	}
}

func (c *ipsecCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *ipsecCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(ctx, re)
	}

	return nil
}

func (c *ipsecCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.Run("/ip/ipsec/policy/print", "?disabled=false", "?dynamic=false", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		return nil, err
	}

	return reply.Re, nil
}

func (c *ipsecCollector) collectForStat(ctx *collectorContext, re *proto.Sentence) {
	srcdst := re.Map["src-address"] + "-" + re.Map["dst-address"]
	comment := re.Map["comment"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(ctx, p, srcdst, comment, re)
	}
}

func (c *ipsecCollector) collectMetricForProperty(ctx *collectorContext, property, srcdst, comment string, re *proto.Sentence) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		var v float64
		var err error
		v, err = strconv.ParseFloat(value, 64)

		switch property {
		case "ph2-state":
			if value == "established" {
				v, err = 1, nil
			} else {
				v, err = 0, nil
			}
		case "active", "invalid":
			if value == "true" {
				v, err = 1, nil
			} else {
				v, err = 0, nil
			}
		case "comment":
			return
		}

		if err != nil {
			ctx.log.Error(
				"error parsing ipsec metric value",
				"srcdst", srcdst,
				"property", property,
				"value", value,
				"err", err,
			)
			return
		}
		ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, srcdst, comment)
	}
}
