package collector

import (
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type interfaceCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newInterfaceCollector() routerOSCollector {
	c := &interfaceCollector{}
	c.init()
	return c
}

func (c *interfaceCollector) init() {
	c.props = []string{"name", "type", "disabled", "comment", "slave", "actual-mtu", "running", "rx-byte", "tx-byte", "rx-packet", "tx-packet", "rx-error", "tx-error", "rx-drop", "tx-drop", "link-downs"}
	labelNames := []string{"interface", "type", "disabled", "comment", "running", "slave"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[5:] {
		c.descriptions[p] = descriptionForPropertyName("interface", p, labelNames)
	}
}

func (c *interfaceCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *interfaceCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *interfaceCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/interface/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		ctx.log.Error(
			"error fetching interface metrics",
			"err", err,
		)
		return nil, err
	}

	return reply.Re, nil
}

func (c *interfaceCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	for _, p := range c.props[5:] {
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *interfaceCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		var (
			v     float64
			vtype prometheus.ValueType
			err   error
		)
		vtype = prometheus.CounterValue

		switch property {
		case "running":
			vtype = prometheus.GaugeValue
			if value == "true" {
				v = 1
			} else {
				v = 0
			}
		case "actual-mtu":
			vtype = prometheus.GaugeValue
			fallthrough
		default:
			v, err = strconv.ParseFloat(value, 64)
			if err != nil {
				ctx.log.Error(
					"error parsing interface metric value",
					"interface", re.Map["name"],
					"property", property,
					"value", value,
					"err", err,
				)
				return
			}
		}
		ctx.ch <- prometheus.MustNewConstMetric(desc, vtype, v,
			re.Map["name"], re.Map["type"], re.Map["disabled"], re.Map["comment"], re.Map["running"], re.Map["slave"])

	}
}
