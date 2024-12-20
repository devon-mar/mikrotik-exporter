package collector

import (
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type capsmanCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newCapsmanCollector() routerOSCollector {
	c := &capsmanCollector{}
	c.init()
	return c
}

func (c *capsmanCollector) init() {
	//"rx-signal", "tx-signal",
	c.props = []string{"interface", "mac-address", "ssid", "uptime", "tx-signal", "rx-signal", "packets", "bytes"}
	labelNames := []string{"interface", "mac_address", "ssid"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[3 : len(c.props)-2] {
		c.descriptions[p] = descriptionForPropertyName("capsman_station", p, labelNames)
	}
	for _, p := range c.props[len(c.props)-2:] {
		c.descriptions["tx_"+p] = descriptionForPropertyName("capsman_station", "tx_"+p, labelNames)
		c.descriptions["rx_"+p] = descriptionForPropertyName("capsman_station", "rx_"+p, labelNames)
	}
}

func (c *capsmanCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *capsmanCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(ctx, re)
	}

	return nil
}

func (c *capsmanCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.Run("/caps-man/registration-table/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		return nil, err
	}

	return reply.Re, nil
}

func (c *capsmanCollector) collectForStat(ctx *collectorContext, re *proto.Sentence) {
	iface := re.Map["interface"]
	mac := re.Map["mac-address"]
	ssid := re.Map["ssid"]

	for _, p := range c.props[3 : len(c.props)-2] {
		c.collectMetricForProperty(ctx, p, iface, mac, ssid, re)
	}
	for _, p := range c.props[len(c.props)-2:] {
		c.collectMetricForTXRXCounters(ctx, p, iface, mac, ssid, re)
	}
}

func (c *capsmanCollector) collectMetricForProperty(ctx *collectorContext, property, iface, mac, ssid string, re *proto.Sentence) {
	if re.Map[property] == "" {
		return
	}
	p := re.Map[property]
	i := strings.Index(p, "@")
	if i > -1 {
		p = p[:i]
	}
	var v float64
	var err error
	if property != "uptime" {
		v, err = strconv.ParseFloat(p, 64)
	} else {
		v, err = parseDuration(p)
	}
	if err != nil {
		ctx.log.Error(
			"error parsing capsman station metric value",
			"property", property,
			"value", re.Map[property],
			"err", err,
		)
		return
	}

	desc := c.descriptions[property]
	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, iface, mac, ssid)
}

func (c *capsmanCollector) collectMetricForTXRXCounters(ctx *collectorContext, property, iface, mac, ssid string, re *proto.Sentence) {
	tx, rx, err := splitStringToFloats(re.Map[property])
	if err != nil {
		ctx.log.Error(
			"error parsing capsman station metric value",
			"property", property,
			"value", re.Map[property],
			"err", err,
		)
		return
	}
	desc_tx := c.descriptions["tx_"+property]
	desc_rx := c.descriptions["rx_"+property]
	ctx.ch <- prometheus.MustNewConstMetric(desc_tx, prometheus.CounterValue, tx, iface, mac, ssid)
	ctx.ch <- prometheus.MustNewConstMetric(desc_rx, prometheus.CounterValue, rx, iface, mac, ssid)
}
