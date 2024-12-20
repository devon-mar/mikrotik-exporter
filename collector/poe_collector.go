package collector

import (
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type poeCollector struct {
	currentDesc *prometheus.Desc
	powerDesc   *prometheus.Desc
	voltageDesc *prometheus.Desc
	props       []string
}

func newPOECollector() routerOSCollector {
	const prefix = "poe"

	labelNames := []string{"interface"}
	return &poeCollector{
		currentDesc: description(prefix, "current", "current in mA", labelNames),
		powerDesc:   description(prefix, "wattage", "Power in W", labelNames),
		voltageDesc: description(prefix, "voltage", "Voltage in V", labelNames),
		props:       []string{"poe-out-current", "poe-out-voltage", "poe-out-power"},
	}
}

func (c *poeCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.currentDesc
	ch <- c.powerDesc
	ch <- c.voltageDesc
}

func (c *poeCollector) collect(ctx *collectorContext) error {
	reply, err := ctx.Run("/interface/ethernet/poe/print", "=.proplist=name")
	if err != nil {
		return err
	}

	ifaces := make([]string, 0)
	for _, iface := range reply.Re {
		n := iface.Map["name"]
		ifaces = append(ifaces, n)
	}

	if len(ifaces) == 0 {
		return nil
	}

	return c.collectPOEMetricsForInterfaces(ctx, ifaces)
}

func (c *poeCollector) collectPOEMetricsForInterfaces(ctx *collectorContext, ifaces []string) error {
	reply, err := ctx.Run("/interface/ethernet/poe/monitor",
		"=numbers="+strings.Join(ifaces, ","),
		"=once=",
		"=.proplist=name,"+strings.Join(c.props, ","))
	if err != nil {
		return err
	}

	for _, se := range reply.Re {
		name, ok := se.Map["name"]
		if !ok {
			continue
		}

		c.collectMetricsForInterface(ctx, name, se)
	}

	return nil
}

func (c *poeCollector) collectMetricsForInterface(ctx *collectorContext, name string, se *proto.Sentence) {
	for _, prop := range c.props {
		v, ok := se.Map[prop]
		if !ok {
			continue
		}
		if v == "" {
			continue
		}
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			ctx.log.Error(
				"error parsing interface poe monitor metric",
				"interface", name,
				"property", prop,
				"err", err,
			)
			return
		}

		ctx.ch <- prometheus.MustNewConstMetric(c.descForKey(prop), prometheus.GaugeValue, value, name)
	}
}

func (c *poeCollector) valueForKey(name, value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

func (c *poeCollector) descForKey(name string) *prometheus.Desc {
	switch name {
	case "poe-out-current":
		return c.currentDesc
	case "poe-out-voltage":
		return c.voltageDesc
	case "poe-out-power":
		return c.powerDesc
	}

	return nil
}
