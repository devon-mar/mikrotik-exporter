package collector

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	uptimeRegex = regexp.MustCompile(`(?:(\d*)w)?(?:(\d*)d)?(?:(\d*)h)?(?:(\d*)m)?(?:(\d*)s)?`)
	uptimeParts = [5]time.Duration{time.Hour * 168, time.Hour * 24, time.Hour, time.Minute, time.Second}
)

type resourceCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newResourceCollector() routerOSCollector {
	c := &resourceCollector{}
	c.init()
	return c
}

func (c *resourceCollector) init() {
	c.props = []string{"free-memory", "total-memory", "cpu-load", "free-hdd-space", "total-hdd-space", "uptime", "board-name", "version"}

	labelNames := []string{"boardname", "version"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = descriptionForPropertyName("system", p, labelNames)
	}
}

func (c *resourceCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *resourceCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(ctx, re)
	}

	return nil
}

func (c *resourceCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.Run("/system/resource/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		return nil, err
	}

	return reply.Re, nil
}

func (c *resourceCollector) collectForStat(ctx *collectorContext, re *proto.Sentence) {
	for _, p := range c.props[:6] {
		c.collectMetricForProperty(ctx, p, re)
	}
}

func (c *resourceCollector) collectMetricForProperty(ctx *collectorContext, property string, re *proto.Sentence) {
	var v float64
	var vtype prometheus.ValueType
	var err error
	//	const boardname = "BOARD"
	//	const version = "3.33.3"

	boardname := re.Map["board-name"]
	version := re.Map["version"]

	if property == "uptime" {
		v, err = parseUptime(re.Map[property])
		vtype = prometheus.CounterValue
	} else {
		if re.Map[property] == "" {
			return
		}
		v, err = strconv.ParseFloat(re.Map[property], 64)
		vtype = prometheus.GaugeValue
	}

	if err != nil {
		ctx.log.Error(
			"error parsing system resource metric value",
			"property", property,
			"value", re.Map[property],
			"err", err,
		)
		return
	}

	desc := c.descriptions[property]
	ctx.ch <- prometheus.MustNewConstMetric(desc, vtype, v, boardname, version)
}

func parseUptime(uptime string) (float64, error) {
	var u time.Duration

	reMatch := uptimeRegex.FindAllStringSubmatch(uptime, -1)

	// should get one and only one match back on the regex
	if len(reMatch) != 1 {
		return 0, fmt.Errorf("invalid uptime value sent to regex")
	}

	for i, match := range reMatch[0] {
		if match != "" && i != 0 {
			v, err := strconv.Atoi(match)
			if err != nil {
				slog.Error(
					"error parsing uptime field value",
					"uptime", uptime,
					"value", match,
					"err", err,
				)
				return float64(0), err
			}
			u += time.Duration(v) * uptimeParts[i-1]
		}
	}
	return u.Seconds(), nil
}
