package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/requests"
	"github.com/fiscalismia/fiscalismia-monitoring/internal/responses"
)

func main() {
	conf, err := config.Load("./targets.yml")
	if err != nil {
		log.Fatalf("Targets could not be loaded from conf: %v", err)
	}

	ctx := context.Background() // TODO: replace with server http request context later

	// Acquire request targets from yaml
	var target *config.Target
	for _, t := range conf.Targets.External {
		if t.Type == "http" {
			target = &t
			fmt.Printf("http target acquired [%v]\n ===> Sending http query en route [%v]\n", target.Name, target.URL)
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
		panic("Could not acquire target from conf.")
	}

	return
}
