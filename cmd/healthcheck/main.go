package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"log/slog"

	"github.com/lmittmann/tint"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/responses"
)

func main() {
	// INIT GLOBAL LOGGING CONFIGURATION
	syslogLevel := slog.LevelDebug
	if os.Getenv("ENVIRONMENT") == "demo" || os.Getenv("ENVIRONMENT") == "prod"  {
		syslogLevel = slog.LevelInfo
	}
	handler := tint.NewHandler(os.Stderr, &tint.Options{
    Level:      syslogLevel,
    TimeFormat: time.Kitchen,  // "3:04PM" — far less noisy than RFC3339
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

	// Acquire request targets from yaml
	var target *config.Target
	for _, t := range conf.Targets.External {
		if t.Type == "http" {
			target = &t
			slog.Debug("http target acquired. Sending http query en route", "name", target.Name, "URL", target.URL)
			// send http requests to targets
			client := requests.CreateClient(conf.GlobalTimeout)
			result := client.QueryHttp(ctx, target)
			if target.X509Verify {
				certValid := client.VerifyTLSCertificate(ctx, target, conf.RootDomain)
				fmt.Printf("%v\n", certValid)
			}

			// ASCII format result from request
			response := responses.ASCII([]requests.Result{result})
			fmt.Printf("%v\n", response)
		}
	}
	if target == nil {
		errorMsg := "Could not acquire target from conf."
		slog.Error(errorMsg)
		os.Exit(1)
	}

	return
}
