package collector

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type dhcpCollector struct {
	leasesActiveCountDesc *prometheus.Desc
}

func (c *dhcpCollector) init() {
	const prefix = "dhcp"

	labelNames := []string{"server"}
	c.leasesActiveCountDesc = description(prefix, "leases_active_count", "number of active leases per DHCP server", labelNames)
}

func newDHCPCollector() routerOSCollector {
	c := &dhcpCollector{}
	c.init()
	return c
}

func (c *dhcpCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.leasesActiveCountDesc
}

func (c *dhcpCollector) collect(ctx *collectorContext) error {
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

func (c *dhcpCollector) fetchDHCPServerNames(ctx *collectorContext) ([]string, error) {
	reply, err := ctx.Run("/ip/dhcp-server/print", "=.proplist=name")
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *dhcpCollector) colllectForDHCPServer(ctx *collectorContext, dhcpServer string) error {
	reply, err := ctx.client.Run("/ip/dhcp-server/lease/print", fmt.Sprintf("?server=%s", dhcpServer), "=active=", "=count-only=")
	if err != nil {
		return fmt.Errorf("/ip/dhcp-server/lease/print server=%s: %w", dhcpServer, err)
	}
	if reply.Done.Map["ret"] == "" {
		return nil
	}
	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		return fmt.Errorf("parse DHCP server %s lease count: %w", dhcpServer, err)
	}

	ctx.ch <- prometheus.MustNewConstMetric(c.leasesActiveCountDesc, prometheus.GaugeValue, v, dhcpServer)
	return nil
}
