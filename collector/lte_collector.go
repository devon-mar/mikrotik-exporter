package collector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type lteCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newLteCollector() routerOSCollector {
	c := &lteCollector{}
	c.init()
	return c
}

func (c *lteCollector) init() {
	c.props = []string{"current-cellid", "primary-band", "ca-band", "rssi", "rsrp", "rsrq", "sinr"}
	labelNames := []string{"interface", "cellid", "primaryband", "caband"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = descriptionForPropertyName("lte_interface", p, labelNames)
	}
}

func (c *lteCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *lteCollector) collect(ctx *collectorContext) error {
	names, err := c.fetchInterfaceNames(ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		err := c.collectForInterface(ctx, n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *lteCollector) fetchInterfaceNames(ctx *collectorContext) ([]string, error) {
	reply, err := ctx.Run("/interface/lte/print", "?disabled=false", "=.proplist=name")
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *lteCollector) collectForInterface(ctx *collectorContext, iface string) error {
	reply, err := ctx.client.Run("/interface/lte/info", fmt.Sprintf("=number=%s", iface), "=once=", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		ctx.log.Error(
			"error fetching interface statistics",
			"interface", iface,
			"err", err,
		)
		return err
	}

	for _, p := range c.props[3:] {
		// there's always going to be only one sentence in reply, as we
		// have to explicitly specify the interface
		c.collectMetricForProperty(ctx, p, iface, reply.Re[0])
	}

	return nil
}

func (c *lteCollector) collectMetricForProperty(ctx *collectorContext, property, iface string, re *proto.Sentence) {
	desc := c.descriptions[property]
	current_cellid := re.Map["current-cellid"]
	// get only band and its width, drop earfcn and phy-cellid info
	primaryband := re.Map["primary-band"]
	if primaryband != "" {
		primaryband = strings.Fields(primaryband)[0]
	}
	caband := re.Map["ca-band"]
	if caband != "" {
		caband = strings.Fields(caband)[0]
	}

	if re.Map[property] == "" {
		return
	}
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		ctx.log.Error(
			"error parsing interface metric value",
			"property", property,
			"interface", iface,
			"err", err,
		)
		return
	}

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, iface, current_cellid, primaryband, caband)
}
