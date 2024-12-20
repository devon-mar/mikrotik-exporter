package collector

import (
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type dhcpLeaseCollector struct {
	props        []string
	descriptions *prometheus.Desc
}

func (c *dhcpLeaseCollector) init() {
	c.props = []string{"active-mac-address", "server", "status", "expires-after", "active-address", "host-name"}

	labelNames := []string{"activemacaddress", "server", "status", "expiresafter", "activeaddress", "hostname"}
	c.descriptions = description("dhcp", "leases_metrics", "number of metrics", labelNames)
}

func newDHCPLCollector() routerOSCollector {
	c := &dhcpLeaseCollector{}
	c.init()
	return c
}

func (c *dhcpLeaseCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.descriptions
}

func (c *dhcpLeaseCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectMetric(ctx, re)
	}

	return nil
}

func (c *dhcpLeaseCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.Run("/ip/dhcp-server/lease/print", "?status=bound", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		return nil, err
	}

	return reply.Re, nil
}

func (c *dhcpLeaseCollector) collectMetric(ctx *collectorContext, re *proto.Sentence) {
	v := 1.0

	f, err := parseDuration(re.Map["expires-after"])
	if err != nil {
		ctx.log.Error(
			"error parsing duration metric value",
			"property", "expires-after",
			"value", re.Map["expires-after"],
			"err", err,
		)
		return
	}

	activemacaddress := re.Map["active-mac-address"]
	server := re.Map["server"]
	status := re.Map["status"]
	activeaddress := re.Map["active-address"]
	// QuoteToASCII because of broken DHCP clients
	hostname := strconv.QuoteToASCII(re.Map["host-name"])

	metric, err := prometheus.NewConstMetric(c.descriptions, prometheus.GaugeValue, v, activemacaddress, server, status, strconv.FormatFloat(f, 'f', 0, 64), activeaddress, hostname)
	if err != nil {
		ctx.log.Error(
			"error parsing dhcp lease",
			"err", err,
		)
		return
	}
	ctx.ch <- metric
}
