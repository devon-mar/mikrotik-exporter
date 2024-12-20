package main

import (
	"bytes"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/prometheus/common/version"

	"mikrotik-exporter/collector"
	"mikrotik-exporter/config"

	"github.com/prometheus/client_golang/prometheus"
)

// single device can be defined via CLI flags, multiple via config file.
var (
	configFile  = flag.String("config", "config.yml", "config file to load")
	logFormat   = flag.String("log-format", "json", "logformat text or json (default json)")
	logLevel    = flag.String("log-level", "info", "log level")
	metricsPath = flag.String("path", "/metrics", "path to answer requests on")
	port        = flag.String("port", ":9436", "port number to listen on")
	ver         = flag.Bool("version", false, "find the version of binary")

	cfg *config.Config

	appVersion = "DEVELOPMENT"
	shortSha   = "0xDEADBEEF"
)

func init() {
	prometheus.MustRegister(version.NewCollector("mikrotik_exporter"))
}

func main() {
	flag.Parse()

	if *ver {
		fmt.Printf("\nVersion:   %s\nShort SHA: %s\n\n", appVersion, shortSha)
		os.Exit(0)
	}

	configureLog()

	c, err := loadConfig()
	if err != nil {
		slog.Error("Could not load config", "err", err)
		os.Exit(3)
	}
	cfg = c

	startServer()
}

func configureLog() {
	var level slog.Level
	err := level.UnmarshalText([]byte(*logLevel))
	if err != nil {
		panic(err)
	}

	handlerOpts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if *logFormat == "text" {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}
	slog.SetDefault(slog.New(handler))
}

func loadConfig() (*config.Config, error) {
	b, err := os.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	return config.Load(bytes.NewReader(b))
}

func startServer() {
	p, err := collector.NewProber(cfg)
	if err != nil {
		slog.Error("error creating prober", "err", err)
		os.Exit(1)
	}

	http.Handle("GET /probe", p)

	http.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>Mikrotik Exporter</title></head>
			<body>
			<h1>Mikrotik Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	slog.Info("Listening", "port", *port)

	err = http.ListenAndServe(*port, nil)
	if err != nil {
		slog.Error("ListenAndServe error", "err", err)
		os.Exit(1)
	}
}
