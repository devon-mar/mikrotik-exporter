package collector

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type firmwareCollector struct {
	description *prometheus.Desc
}

func newFirmwareCollector() routerOSCollector {
	c := &firmwareCollector{}
	c.init()
	return c
}

func (c *firmwareCollector) init() {
	labelNames := []string{"name", "disabled", "version", "build_time"}
	c.description = description("system", "package", "system packages version", labelNames)
}

func (c *firmwareCollector) describe(ch chan<- *prometheus.Desc) {
	ch <- c.description
}

func (c *firmwareCollector) collect(ctx *collectorContext) error {
	reply, err := ctx.Run("/system/package/getall")
	if err != nil {
		return err
	}

	pkgs := reply.Re

	for _, pkg := range pkgs {
		v := 1.0
		if strings.EqualFold(pkg.Map["disabled"], "true") {
			v = 0.0
		}
		ctx.ch <- prometheus.MustNewConstMetric(c.description, prometheus.GaugeValue, v, pkg.Map["name"], pkg.Map["disabled"], pkg.Map["version"], pkg.Map["build-time"])
	}

	return nil
}
