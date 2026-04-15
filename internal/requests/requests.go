package requests

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fiscalismia/fiscalismia-monitoring/internal/config"
)

const (
	READ_CAPACITY_BYTES int64 = 1024
)

type Result struct {
	Name       string
	URL        string
	Host       string
	Type       string
	StatusCode int
	Body       string
	Latency    time.Duration
	Err        error
}

type Client struct {
	client *http.Client
}

func CreateClient(globalTimeout time.Duration) *Client {
	return &Client{
		client: &http.Client{
			Timeout: globalTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// TODO: enforce where possible
					InsecureSkipVerify: true,
				},
			},
		},
	}
}

// Method on the Client struct
func (c *Client) QueryHttp(t *config.Target) Result {
	start := time.Now()
	latency := time.Since(start)

	resp, err := c.client.Get(t.URL)
	if err != nil {
		log.Fatalf("request to %s failed: %v", t.Name, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, READ_CAPACITY_BYTES))
	if err != nil {
		return Result{Name: t.Name, URL: t.URL, Type: t.Type, StatusCode: resp.StatusCode, Latency: latency, Err: err}
	}

	return Result{
		Name:       t.Name,
		URL:        t.URL,
		Type:       t.Type,
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Latency:    latency,
	}
}
