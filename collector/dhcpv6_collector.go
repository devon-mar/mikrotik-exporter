package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type dhcpv6Collector struct {
	bindingCountDesc *prometheus.Desc
}

func newDHCPv6Collector() routerOSCollector {
	c := &dhcpv6Collector{}
	c.init()
	return c
}

func (c *dhcpv6Collector) init() {
	const prefix = "dhcpv6"

	labelNames := []string{"server"}
	c.bindingCountDesc = description(prefix, "binding_count", "number of active bindings per DHCPv6 server", labelNames)
}

func (c *dhcpv6Collector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.bindingCountDesc
}

func (c *dhcpv6Collector) collect(ctx *collectorContext) error {
	names, err := c.fetchDHCPServerNames(ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		err := c.colllectForDHCPServer(ctx, n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *dhcpv6Collector) fetchDHCPServerNames(ctx *collectorContext) ([]string, error) {
	reply, err := ctx.Run("/ipv6/dhcp-server/print", "=.proplist=name")
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *dhcpv6Collector) colllectForDHCPServer(ctx *collectorContext, dhcpServer string) error {
	reply, err := ctx.client.Run("/ipv6/dhcp-server/binding/print", fmt.Sprintf("?server=%s", dhcpServer), "=count-only=")
	if err != nil {
		ctx.log.Error(
			"error fetching DHCPv6 binding counts",
			"dhcpv6_server", dhcpServer,
			"err", err,
		)
		return err
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		ctx.log.Error(
			"error parsing DHCPv6 binding counts",
			"dhcpv6_server", dhcpServer,
			"err", err,
		)
		return err
	}

	ctx.ch <- prometheus.MustNewConstMetric(c.bindingCountDesc, prometheus.GaugeValue, v, dhcpServer)
	return nil
}
