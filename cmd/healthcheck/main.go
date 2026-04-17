package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/responses"
)

func main() {
	// INIT GLOBAL LOGGING CONFIGURATION
	syslogLevel := slog.LevelDebug
	if os.Getenv("ENVIRONMENT") == "demo" || os.Getenv("ENVIRONMENT") == "prod" {
		syslogLevel = slog.LevelInfo
	}
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      syslogLevel,
		TimeFormat: time.Kitchen, // "3:04PM" — far less noisy than RFC3339
	})
	slog.SetDefault(slog.New(handler))
	slog.Debug("hello", "debug", "derp")
	slog.Info("hello", "info", "derp")

	conf, err := config.Load("./targets.yml")
	if err != nil {
		slog.Error("could not load config", "path", "./targets.yml", "err", err)
		os.Exit(1)
	}

	ctx := context.Background() // TODO: replace with server http request context later

	// Acquire external (publicly reachable) request targets from yaml
	var target *config.Target
	for _, t := range conf.Targets.External {
		target = &t
		var result requests.Result
		client := requests.CreateClient(conf.GlobalTimeout)
		switch target.Type {
		case "http":
			slog.Debug("[http] target acquired. Sending http query en route", "name", target.Name, "URL", target.URL)
			// send http requests to targets
			result = client.QueryHTTP(ctx, target)
			if target.X509Verify {
				x509Verify := client.VerifyTLSCertificate(ctx, target, conf.RootDomain)
				result.X509Info = x509Verify
			}

			// ASCII format result from request
			response := responses.ASCII([]requests.Result{result})
			fmt.Printf("%v\n", response)
		case "tcp":
			slog.Debug("[tcp] target acquired. Sending tcp query to host", "name", target.Name, "host", target.Host)
			result = client.QueryTCP(ctx, target)
		case "icmp":
			slog.Debug("[icmp] target acquired. Try raw ICMP socket conn query", "name", target.Name, "host", target.Host)
			result = client.QueryICMP(ctx, target)
		}
		// ASCII format result from request
		response := responses.ASCII([]requests.Result{result})
		fmt.Printf("%v\n", response)
	}
	if target == nil {
		errorMsg := "Could not acquire target from conf."
		slog.Error(errorMsg)
		os.Exit(1)
	}
}
