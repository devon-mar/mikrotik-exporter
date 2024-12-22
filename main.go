package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/prometheus/client_golang/prometheus/collectors/version"

	"mikrotik-exporter/collector"
	"mikrotik-exporter/config"

	"github.com/prometheus/client_golang/prometheus"
)

// single device can be defined via CLI flags, multiple via config file.
var (
	configFile = flag.String("config", "config.yml", "config file to load")
	logFormat  = flag.String("log-format", "text", "logformat text or json (default json)")
	logLevel   = flag.String("log-level", "info", "log level")
	addr       = flag.String("port", ":9436", "port number to listen on")
	ver        = flag.Bool("version", false, "find the version of binary")

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

	mux := http.NewServeMux()

	mux.Handle("GET /probe", p)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>Mikrotik Exporter</title></head>
			<body>
			<h1>Mikrotik Exporter</h1>
			</body>
			</html>`))
	})

	// Modified from example https://pkg.go.dev/net/http#Server.Close
	srv := http.Server{
		Addr:    *addr,
		Handler: mux,
	}
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	slog.Info("Listening on", "port", *addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}
