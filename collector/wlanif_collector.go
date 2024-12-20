package collector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type wlanIFCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newWlanIFCollector() routerOSCollector {
	c := &wlanIFCollector{}
	c.init()
	return c
}

func (c *wlanIFCollector) init() {
	c.props = []string{"channel", "registered-clients", "noise-floor", "overall-tx-ccq"}
	labelNames := []string{"interface", "channel"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = descriptionForPropertyName("wlan_interface", p, labelNames)
	}
}

func (c *wlanIFCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *wlanIFCollector) collect(ctx *collectorContext) error {
	names, err := c.fetchInterfaceNames(ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		err := c.collectForInterface(n, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *wlanIFCollector) fetchInterfaceNames(ctx *collectorContext) ([]string, error) {
	reply, err := ctx.Run("/interface/wireless/print", "?disabled=false", "=.proplist=name")
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *wlanIFCollector) collectForInterface(iface string, ctx *collectorContext) error {
	reply, err := ctx.client.Run("/interface/wireless/monitor", fmt.Sprintf("=numbers=%s", iface), "=once=", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		ctx.log.Error(
			"error fetching interface statistics",
			"interface", iface,
			"err", err,
		)
		return err
	}

	for _, p := range c.props[1:] {
		// there's always going to be only one sentence in reply, as we
		// have to explicitly specify the interface
		c.collectMetricForProperty(p, iface, reply.Re[0], ctx)
	}

	return nil
}

func (c *wlanIFCollector) collectMetricForProperty(property, iface string, re *proto.Sentence, ctx *collectorContext) {
	desc := c.descriptions[property]
	channel := re.Map["channel"]
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

	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, iface, channel)
}
