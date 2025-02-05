package collector

import (
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type monitorCollector struct {
	props        []string // props from monitor, can add other ether props later if needed
	descriptions map[string]*prometheus.Desc
}

func newMonitorCollector() routerOSCollector {
	c := &monitorCollector{}
	c.init()
	return c
}

func (c *monitorCollector) init() {
	c.props = []string{"status", "rate", "full-duplex"}
	labelNames := []string{"interface"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = descriptionForPropertyName("monitor", p, labelNames)
	}
}

func (c *monitorCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *monitorCollector) collect(ctx *collectorContext) error {
	reply, err := ctx.Run("/interface/ethernet/print", "=.proplist=name")
	if err != nil {
		return err
	}

	eths := make([]string, len(reply.Re))
	for idx, eth := range reply.Re {
		eths[idx] = eth.Map["name"]
	}

	return c.collectForMonitor(ctx, eths)
}

func (c *monitorCollector) collectForMonitor(ctx *collectorContext, eths []string) error {
	reply, err := ctx.Run("/interface/ethernet/monitor",
		"=numbers="+strings.Join(eths, ","),
		"=once=",
		"=.proplist=name,"+strings.Join(c.props, ","))
	if err != nil {
		ctx.log.Error(
			"error fetching ethernet monitor info",
			"err", err,
		)
		return err
	}

	for _, e := range reply.Re {
		c.collectMetricsForEth(ctx, e.Map["name"], e)
	}

	return nil
}

func (c *monitorCollector) collectMetricsForEth(ctx *collectorContext, name string, se *proto.Sentence) {
	for _, prop := range c.props {
		v, ok := se.Map[prop]
		if !ok {
			continue
		}

		value := float64(c.valueForProp(prop, v))

		ctx.ch <- prometheus.MustNewConstMetric(c.descriptions[prop], prometheus.GaugeValue, value, name)

	}
}

func (c *monitorCollector) valueForProp(name, value string) int {
	switch {
	case name == "status":
		return func(v string) int {
			if v == "link-ok" {
				return 1
			}
			return 0
		}(value)
	case name == "rate":
		return func(v string) int {
			switch {
			case v == "10Mbps":
				return 10
			case v == "100Mbps":
				return 100
			case v == "1Gbps":
				return 1000
			case v == "10Gbps":
				return 10000
			}
			return 0
		}(value)
	case name == "full-duplex":
		return func(v string) int {
			if v == "true" {
				return 1
			}
			return 0
		}(value)
	default:
		return 0
	}
}
