package collector

import (
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type wlanSTACollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newWlanSTACollector() routerOSCollector {
	c := &wlanSTACollector{}
	c.init()
	return c
}

func (c *wlanSTACollector) init() {
	c.props = []string{"interface", "mac-address", "signal-to-noise", "signal-strength", "packets", "bytes", "frames"}
	labelNames := []string{"interface", "mac_address"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[:len(c.props)-3] {
		c.descriptions[p] = descriptionForPropertyName("wlan_station", p, labelNames)
	}
	for _, p := range c.props[len(c.props)-3:] {
		c.descriptions["tx_"+p] = descriptionForPropertyName("wlan_station", "tx_"+p, labelNames)
		c.descriptions["rx_"+p] = descriptionForPropertyName("wlan_station", "rx_"+p, labelNames)
	}
}

func (c *wlanSTACollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *wlanSTACollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(ctx, re)
	}

	return nil
}

func (c *wlanSTACollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.Run("/interface/wireless/registration-table/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		return nil, err
	}

	return reply.Re, nil
}

func (c *wlanSTACollector) collectForStat(ctx *collectorContext, re *proto.Sentence) {
	iface := re.Map["interface"]
	mac := re.Map["mac-address"]

	for _, p := range c.props[2 : len(c.props)-3] {
		c.collectMetricForProperty(ctx, p, iface, mac, re)
	}
	for _, p := range c.props[len(c.props)-3:] {
		c.collectMetricForTXRXCounters(ctx, p, iface, mac, re)
	}
}

func (c *wlanSTACollector) collectMetricForProperty(ctx *collectorContext, property, iface, mac string, re *proto.Sentence) {
	if re.Map[property] == "" {
		return
	}
	p := re.Map[property]
	i := strings.Index(p, "@")
	if i > -1 {
		p = p[:i]
	}
	v, err := strconv.ParseFloat(p, 64)
	if err != nil {
		ctx.log.Error(
			"error parsing wlan station metric value",
			"property", property,
			"value", re.Map[property],
			"err", err,
		)
		return
	}

	desc := c.descriptions[property]
	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, iface, mac)
}

func (c *wlanSTACollector) collectMetricForTXRXCounters(ctx *collectorContext, property, iface, mac string, re *proto.Sentence) {
	tx, rx, err := splitStringToFloats(re.Map[property])
	if err != nil {
		ctx.log.Error(
			"error parsing wlan station metric value",
			"property", property,
			"value", re.Map[property],
			"err", err,
		)
		return
	}
	desc_tx := c.descriptions["tx_"+property]
	desc_rx := c.descriptions["rx_"+property]
	ctx.ch <- prometheus.MustNewConstMetric(desc_tx, prometheus.CounterValue, tx, iface, mac)
	ctx.ch <- prometheus.MustNewConstMetric(desc_rx, prometheus.CounterValue, rx, iface, mac)
}
