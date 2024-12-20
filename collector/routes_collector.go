package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type routesCollector struct {
	protocols         []string
	countDesc         *prometheus.Desc
	countProtocolDesc *prometheus.Desc
}

func newRoutesCollector() routerOSCollector {
	c := &routesCollector{}
	c.init()
	return c
}

func (c *routesCollector) init() {
	const prefix = "routes"
	labelNames := []string{"ip_version"}
	c.countDesc = description(prefix, "total_count", "number of routes in RIB", labelNames)
	c.countProtocolDesc = description(prefix, "protocol_count", "number of routes per protocol in RIB", append(labelNames, "protocol"))

	c.protocols = []string{"bgp", "static", "ospf", "dynamic", "connect", "rip"}
}

func (c *routesCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.countDesc
	ch <- c.countProtocolDesc
}

func (c *routesCollector) collect(ctx *collectorContext) error {
	err := c.colllectForIPVersion("4", "ip", ctx)
	if err != nil {
		return err
	}

	return c.colllectForIPVersion("6", "ip", ctx)
}

func (c *routesCollector) colllectForIPVersion(ipVersion, topic string, ctx *collectorContext) error {
	err := c.colllectCount(ipVersion, topic, ctx)
	if err != nil {
		return err
	}

	for _, p := range c.protocols {
		err := c.colllectCountProtcol(ipVersion, topic, p, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *routesCollector) colllectCount(ipVersion, topic string, ctx *collectorContext) error {
	reply, err := ctx.client.Run(fmt.Sprintf("/%s/route/print", topic), "?disabled=false", "=count-only=")
	if err != nil {
		ctx.log.Error(
			"error fetching routes metrics",
			"ip_version", ipVersion,
			"topic", topic,
			"err", err,
		)
		return err
	}
	if reply.Done.Map["ret"] == "" {
		return nil
	}
	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		ctx.log.Error(
			"error parsing routes metrics",
			"ip_version", ipVersion,
			"err", err,
		)
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(c.countDesc, prometheus.GaugeValue, v, ipVersion)
	return nil
}

func (c *routesCollector) colllectCountProtcol(ipVersion, topic, protocol string, ctx *collectorContext) error {
	reply, err := ctx.client.Run(fmt.Sprintf("/%s/route/print", topic), "?disabled=false", fmt.Sprintf("?%s", protocol), "=count-only=")
	if err != nil {
		ctx.log.Error(
			"error fetching routes metrics",
			"ip_version", ipVersion,
			"protocol", protocol,
			"err", err,
		)
		return err
	}
	if reply.Done.Map["ret"] == "" {
		return nil
	}
	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		ctx.log.Error(
			"error parsing routes metrics",
			"ip_version", ipVersion,
			"protocol", protocol,
			"err", err,
		)
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(c.countProtocolDesc, prometheus.GaugeValue, v, ipVersion, protocol)
	return nil
}
