package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"zerog-exporter/config"
	"zerog-exporter/collector"
	"zerog-exporter/rpc"
)

func main() {
	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	var logLevel slog.Level
	switch cfg.Logging.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: logLevel}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))

	for i := range cfg.Chains {
		chain := &cfg.Chains[i]
		if chain.AutoDetect {
			logger.Info("Auto-detection enabled for chain", "chain_id", chain.ChainID)
		}
	}

	registry := prometheus.NewRegistry()

	for _, chain := range cfg.Chains {
		client := rpc.NewClient(chain.RPC, chain.API, chain.WebSocket)
		unifiedCollector := collector.NewUnifiedCollector(client, &chain, &cfg.Ethereum, cfg.Prometheus.Server)
		registry.MustRegister(unifiedCollector)
	}

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logger.Info("Starting server", "address", cfg.ListenAddress)
	if err := http.ListenAndServe(cfg.ListenAddress, nil); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down gracefully...")
}