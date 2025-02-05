package collector

import (
	"strconv"
	"strings"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

type w60gInterfaceCollector struct {
	frequencyDesc         *prometheus.Desc
	txMCSDesc             *prometheus.Desc
	txPHYRateDesc         *prometheus.Desc
	signalDesc            *prometheus.Desc
	rssiDesc              *prometheus.Desc
	txSectorDesc          *prometheus.Desc
	txDistanceDesc        *prometheus.Desc
	txPacketErrorRateDesc *prometheus.Desc
	props                 []string
}

func (c *w60gInterfaceCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.frequencyDesc
	ch <- c.txMCSDesc
	ch <- c.txPHYRateDesc
	ch <- c.signalDesc
	ch <- c.rssiDesc
	ch <- c.txSectorDesc
	ch <- c.txDistanceDesc
	ch <- c.txPacketErrorRateDesc
}

func (c *w60gInterfaceCollector) collect(ctx *collectorContext) error {
	reply, err := ctx.Run("/interface/w60g/print", "=.proplist=name")
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

	return c.collectw60gMetricsForInterfaces(ctx, ifaces)
}

func (c *w60gInterfaceCollector) collectw60gMetricsForInterfaces(ctx *collectorContext, ifaces []string) error {
	reply, err := ctx.client.Run("/interface/w60g/monitor",
		"=numbers="+strings.Join(ifaces, ","),
		"=once=",
		"=.proplist=name,"+strings.Join(c.props, ","))
	if err != nil {
		ctx.log.Error(
			"error fetching w60g interface monitor metrics",
			"err", err,
		)
		return err
	}
	for _, se := range reply.Re {
		name, ok := se.Map["name"]
		if !ok {
			continue
		}

		c.collectMetricsForw60gInterface(ctx, name, se)
	}

	return nil
}

func (c *w60gInterfaceCollector) collectMetricsForw60gInterface(ctx *collectorContext, name string, se *proto.Sentence) {
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
				"error parsing w60g interface monitor metric",
				"interface", name,
				"property", prop,
				"err", err,
			)
			return
		}

		ctx.ch <- prometheus.MustNewConstMetric(c.descForKey(prop), prometheus.GaugeValue, value, name)
	}
}

func neww60gInterfaceCollector() routerOSCollector {
	const prefix = "w60ginterface"

	labelNames := []string{"interface"}
	return &w60gInterfaceCollector{
		frequencyDesc:         description(prefix, "frequency", "frequency of tx in MHz", labelNames),
		txMCSDesc:             description(prefix, "txMCS", "TX MCS", labelNames),
		txPHYRateDesc:         description(prefix, "txPHYRate", "PHY Rate in bps", labelNames),
		signalDesc:            description(prefix, "signal", "Signal quality in %", labelNames),
		rssiDesc:              description(prefix, "rssi", "Signal RSSI in dB", labelNames),
		txSectorDesc:          description(prefix, "txSector", "TX Sector", labelNames),
		txDistanceDesc:        description(prefix, "txDistance", "Distance to remote", labelNames),
		txPacketErrorRateDesc: description(prefix, "txPacketErrorRate", "TX Packet Error Rate", labelNames),
		props:                 []string{"signal", "rssi", "tx-mcs", "frequency", "tx-phy-rate", "tx-sector", "distance", "tx-packet-error-rate"},
	}
}

func (c *w60gInterfaceCollector) descForKey(name string) *prometheus.Desc {
	switch name {
	case "signal":
		return c.signalDesc
	case "rssi":
		return c.rssiDesc
	case "tx-mcs":
		return c.txMCSDesc
	case "tx-phy-rate":
		return c.txPHYRateDesc
	case "frequency":
		return c.frequencyDesc
	case "tx-sector":
		return c.txSectorDesc
	case "distance":
		return c.txDistanceDesc
	case "tx-packet-error-rate":
		return c.txPacketErrorRateDesc
	}

	return nil
}
