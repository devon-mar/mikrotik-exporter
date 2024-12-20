package collector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"mikrotik-exporter/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	paramTarget = "target"
	paramModule = "module"
)

type proberModule struct {
	c       *collector
	timeout time.Duration
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

func readCertificate(file string) (*x509.Certificate, error) {
	const pemBlockCert = "CERTIFICATE"

	certBytes, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("ReadFile: %w", err)
	}

	block, _ := pem.Decode(certBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM data found.")
	}

	if block.Type != pemBlockCert {
		return nil, fmt.Errorf("unexpected block type: %s", block.Type)
	}

	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing certificate: %w", err)
	}

	return c, nil
}

func NewProber(c *config.Config) (http.Handler, error) {
	p := &Prober{modules: make(map[string]proberModule, len(c.Modules))}

	for name, m := range c.Modules {
		timeout := DefaultTimeout
		if m.Timeout != 0 {
			timeout = time.Duration(m.Timeout)
		}

		var rootCAs *x509.CertPool
		if m.CACert != "" {
			cert, err := readCertificate(m.CACert)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", m.CACert, err)
			}
			rootCAs = x509.NewCertPool()
			rootCAs.AddCert(cert)
		}

		tlsCfg := &tls.Config{
			InsecureSkipVerify: m.InsecureTLS,
			RootCAs:            rootCAs,
		}

		p.modules[name] = proberModule{
			timeout: timeout,
			c: &collector{
				tlsCfg:       tlsCfg,
				collectors:   collectorList(m.Features),
				usernameFile: m.UsernameFile,
				passwordFile: m.PasswordFile,
				usernameStr:  m.Username,
				passwordStr:  m.Password,
			},
		}
	}

	return p, nil
}

// ServeHTTP implements http.Handler
func (p *Prober) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "no target", http.StatusBadRequest)
		return
	}

	moduleName := r.URL.Query().Get("module")

	module, ok := p.modules[moduleName]
	if !ok {
		http.Error(w, "invalid module", http.StatusBadRequest)
		return
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(&proberCollector{
		c:       module.c,
		target:  target,
		timeout: module.timeout,
	})

	promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog:      log.Default(),
		ErrorHandling: promhttp.ContinueOnError,
	}).ServeHTTP(w, r)
}

type proberCollector struct {
	c       *collector
	target  string
	timeout time.Duration
}

// Collect implements prometheus.Collector
func (pc *proberCollector) Collect(c chan<- prometheus.Metric) {
	// https://github.com/prometheus/client_golang/issues/1538
	ctx, cancel := context.WithTimeout(context.Background(), pc.timeout)
	defer cancel()

	pc.c.collectForDevice(ctx, pc.target, c)
}

// Describe implements prometheus.Collector
func (pc *proberCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc

	for _, co := range pc.c.collectors {
		co.describe(ch)
	}
}
