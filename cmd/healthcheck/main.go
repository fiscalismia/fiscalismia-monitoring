package main

import (
	// "crypto/tls"
	"fmt"
	// "io"
	"log"
	// "net/http"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
)

func main() {
	conf, err := config.Load("configs/targets.yml")
	if err != nil {
		log.Fatal(err) // log.Fatal prints and calls os.Exit(1)
	}

	// var target config.Target
	for _, t := range conf.Targets.External {
		fmt.Printf("found External target %#v\n", t)
		// if t.Name == "Demo Backend" {
		// 	target = t
		// 	fmt.Printf("Target acquired %v\n", target)
		// 	break
		// }
	}
	return
}
