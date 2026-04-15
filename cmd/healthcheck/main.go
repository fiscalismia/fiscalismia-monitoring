package main

import (
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

	// Acquire request targets from yaml
	var target *config.Target
	for _, t := range conf.Targets.External {
		if t.Name == "Demo Backend" {
			target = &t
			fmt.Printf("Target acquired %v\n", target)
			break
		}
	}
	if target == nil {
		panic("Could not acquire target from conf.")
	}

	// send http requests to targets
	client := requests.CreateClient(conf.GlobalTimeout)
	result := client.QueryHttp(target)

	// ASCII format result from request
	response := responses.ASCII([]requests.Result{result})
	fmt.Printf("%v\n", response)

	return
}
