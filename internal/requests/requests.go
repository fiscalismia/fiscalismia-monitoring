package requests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
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

type X509CertificateValidity struct {
	IsValid bool
	Err     error
}

type Client struct {
	client *http.Client
}

// Public Method of the Client struct
func (c *Client) QueryHttp(ctx context.Context, t *config.Target) Result {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		return Result{Name: t.Name, URL: t.URL, Type: t.Type, Err: err}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return Result{Name: t.Name, URL: t.URL, Type: t.Type, Err: err}
	}
	defer resp.Body.Close()

	latency := time.Since(start)
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

func (c *Client) VerifyTLSCertificate(ctx context.Context, t *config.Target, rd string) X509CertificateValidity {
	// sanity check for target URL
	if !strings.HasPrefix(t.URL, "https://") {
		return X509CertificateValidity{IsValid: false, Err: errors.New("URL does not contain https:// protocol.")}
	}

	tlsUrl, err := cleanTlsURL(t.URL, rd)
	if err != nil {
		return X509CertificateValidity{IsValid: false, Err: err}
	}

	config := &tls.Config{
		InsecureSkipVerify: false,
	}

	// Establish a TCP connection first
	conn, err := tls.Dial("tcp", tlsUrl, config)
	if err != nil {
		return X509CertificateValidity{IsValid: false, Err: err}
	}
	defer conn.Close()

	// Perform SSL/TLS handshake
	err = conn.Handshake()
	if err != nil {
		return X509CertificateValidity{IsValid: false, Err: err}
	}

	// Retrieve the peer certificates
	certs := conn.ConnectionState().PeerCertificates

	// Iterate and validate each certificate in the chain
	for _, cert := range certs {
		_, err := cert.Verify(x509.VerifyOptions{})
		if err != nil {
			return X509CertificateValidity{IsValid: false, Err: err}
		}
	}

	log.Printf("Successfully verified host: %s certificates", t.URL)
	return X509CertificateValidity{
		IsValid: true,
	}
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

// strips any trailing path within the fqdn
func cleanTlsURL(url string, rootDomain string) (string, error) {
	proto := "https://"
	idx := strings.Index(url, rootDomain)
	if idx == 0 {
		return "", errors.New("Target.URL does not contain rootDomain")
	}
	// strips https:// and anything trailing .com
	baseUrl := url[len(proto):idx+len(rootDomain)]
	return baseUrl + ":443", nil
}
