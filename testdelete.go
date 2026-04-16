package main

import (
	"fmt"
	"strings"
)

func main() {
	URL := "https://demo.fiscalismia.com/api/fiscalismia/hc"
	sub := "fiscalismia.com"
	idx := strings.Index(URL, "fiscalismia.com")
	baseUrl := URL[:idx+len(sub)]
	fmt.Printf("%v\n", idx)
	fmt.Printf("%v\n", strings.Join(baseUrl, "443"))
}
