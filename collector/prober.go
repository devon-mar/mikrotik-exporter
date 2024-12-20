package collector

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"mikrotik-exporter/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	paramTarget = "target"
	paramModule = "module"
	portMax     = 65535
)

type proberModule struct {
	c        *collector
	username string
	password string
}

type Prober struct {
	modules map[string]proberModule
}

func collectorList(f config.Features) []routerOSCollector {
	c := []routerOSCollector{}

	if f.BGP {
		c = append(c, newBGPCollector())
	}

	if f.Conntrack {
		c = append(c, newConntrackCollector())
	}

	if f.DHCP {
		c = append(c, newDHCPCollector())
	}

	if f.DHCPL {
		c = append(c, newDHCPLCollector())
	}

	if f.DHCPv6 {
		c = append(c, newDHCPv6Collector())
	}

	if f.Firmware {
		c = append(c, newFirmwareCollector())
	}

	if f.Health {
		c = append(c, newhealthCollector())
	}

	if f.Routes {
		c = append(c, newRoutesCollector())
	}

	if f.POE {
		c = append(c, newPOECollector())
	}

	if f.Pools {
		c = append(c, newPoolCollector())
	}

	if f.Optics {
		c = append(c, newOpticsCollector())
	}

	if f.W60G {
		c = append(c, neww60gInterfaceCollector())
	}

	if f.WlanSTA {
		c = append(c, newWlanSTACollector())
	}

	if f.Capsman {
		c = append(c, newCapsmanCollector())
	}

	if f.WlanIF {
		c = append(c, newWlanIFCollector())
	}

	if f.Monitor {
		c = append(c, newMonitorCollector())
	}

	if f.Ipsec {
		c = append(c, newIpsecCollector())
	}

	if f.Lte {
		c = append(c, newLteCollector())
	}

	if f.Netwatch {
		c = append(c, newNetwatchCollector())
	}

	return c
}

func NewProber(c *config.Config) http.Handler {
	p := &Prober{modules: make(map[string]proberModule, len(c.Modules))}

	for name, m := range c.Modules {
		timeout := DefaultTimeout
		if m.Timeout != 0 {
			timeout = time.Duration(m.Timeout)
		}
		p.modules[name] = proberModule{
			username: m.Username,
			password: m.Password,
			c: &collector{
				collectors:  collectorList(m.Features),
				timeout:     timeout,
				enableTLS:   m.EnableTLS,
				insecureTLS: m.InsecureTLS,
			},
		}
	}

	return p
}

// ServeHTTP implements http.Handler
func (p *Prober) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	target := r.URL.Query().Get("target")
	moduleName := r.URL.Query().Get("module")

	module, ok := p.modules[moduleName]
	if !ok {
		http.Error(w, "invalid module", http.StatusBadRequest)
		return
	}

	host, portStr, err := net.SplitHostPort(target)
	if err != nil || host == "" || portStr == "" {
		http.Error(w, "invalid target", http.StatusBadRequest)
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > portMax {
		http.Error(w, "invalid port", http.StatusBadRequest)
		return
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(&proberCollector{module.c, config.Device{
		Name:     host,
		Port:     portStr,
		Address:  host,
		User:     module.username,
		Password: module.password,
	}})

	promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog:      log.Default(),
		ErrorHandling: promhttp.ContinueOnError,
	}).ServeHTTP(w, r)
}

type proberCollector struct {
	c *collector
	d config.Device
}

// Collect implements prometheus.Collector
func (pc *proberCollector) Collect(c chan<- prometheus.Metric) {
	pc.c.collectForDevice(pc.d, c)
}

// Describe implements prometheus.Collector
func (pc *proberCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc

	for _, co := range pc.c.collectors {
		co.describe(ch)
	}
}
