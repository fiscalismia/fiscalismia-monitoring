package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
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
	client := &http.Client{
		Timeout: conf.GlobalTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get(target.URL)
	if err != nil {
		log.Fatalf("request to %s failed: %v", target.Name, err)
	}
	defer resp.Body.Close()

	// parse http responses received from targets
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("read body: %v", err)
	}
	fmt.Printf("[%d] %s → %s\n", resp.StatusCode, target.Name, string(body))

	return
}
