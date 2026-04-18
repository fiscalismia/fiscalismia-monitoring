package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmittmann/tint"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/responses"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/version"
)

func main() {
	///// INIT GLOBAL LOGGING CONFIGURATION
	syslogLevel := slog.LevelDebug
	if os.Getenv("ENVIRONMENT") == "demo" || os.Getenv("ENVIRONMENT") == "prod" {
		syslogLevel = slog.LevelInfo
	}
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      syslogLevel,
		TimeFormat: time.Kitchen, // "3:04PM" — far less noisy than RFC3339
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("starting healthcheck", "version", version.Version, "commit", version.Commit, "buildTime", version.BuildTime)

	///// LOAD CONFIG
	conf, err := config.Load("./targets.yml")
	if err != nil {
		slog.Error("could not load config", "path", "./targets.yml", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	///// ACQUIRE EXTERNAL TARGETS (publicly reachable) from targets.yaml
	var target *config.Target
	var results []requests.Result
	for _, t := range conf.Targets.External {
		target = &t
		var result requests.Result
		client := requests.CreateClient(conf.GlobalTimeout)
		switch target.Type {
		case "http":
			slog.Debug("[http] target acquired. Sending http query en route", "name", target.Name, "URL", target.URL)
			result = client.QueryHTTP(ctx, target)
			if target.X509Verify {
				x509Verify := client.VerifyTLSCertificate(ctx, target, conf.RootDomain)
				result.X509Info = x509Verify
			}
		case "tcp":
			slog.Debug("[tcp] target acquired. Sending tcp query to host", "name", target.Name, "host", target.Host)
			result = client.QueryTCP(ctx, target)
		case "icmp":
			slog.Debug("[icmp] target acquired. Try raw ICMP socket conn query", "name", target.Name, "host", target.Host)
			result = client.QueryICMP(ctx, target)
		}
		results = append(results, result)
	}

	// ASCII format result from request
	response := responses.ASCII(results)
	fmt.Printf("%v\n", response)
}
