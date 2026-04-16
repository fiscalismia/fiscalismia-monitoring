package requests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
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
	Expires time.Time
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
		return X509CertificateValidity{IsValid: false, Expires: time.Now(), Err: err}
	}

	config := &tls.Config{
		InsecureSkipVerify: false,
	}

	// Establish a TCP connection first
	conn, err := tls.Dial("tcp", tlsUrl, config)
	if err != nil {
		return X509CertificateValidity{IsValid: false, Expires: time.Now(), Err: err}
	}
	defer conn.Close()

	// Perform SSL/TLS handshake
	err = conn.Handshake()
	if err != nil {
		return X509CertificateValidity{IsValid: false, Expires: time.Now(), Err: err}
	}

	// Retrieve the peer certificates
	certs := conn.ConnectionState().PeerCertificates

	// Iterate and validate each certificate in the chain
	for _, cert := range certs {
		fmt.Printf("%v\n", formatCert(cert))
	}

	log.Printf("Successfully verified host: %s certificates", t.URL)
	return X509CertificateValidity{
		IsValid: true,
		Expires: time.Now(),
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

// strips the protocol and any trailing path within the fqdn
func cleanTlsURL(url string, rootDomain string) (string, error) {
	proto := "https://"
	idx := strings.Index(url, rootDomain)
	if idx == -1 {
		return "", errors.New("Target.URL does not contain rootDomain")
	}
	// strips https:// and anything trailing .com
	baseUrl := url[len(proto) : idx+len(rootDomain)]
	return baseUrl + ":443", nil
}

// pretty prints TLS Certificate fields for detailed analysis
func formatCert(cert *x509.Certificate) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Subject:        %s\n", cert.Subject.CommonName)
	fmt.Fprintf(&b, "Issuer:         %s\n", cert.Issuer.CommonName)
	fmt.Fprintf(&b, "Serial:         %s\n", cert.SerialNumber.Text(16))
	fmt.Fprintf(&b, "Valid From:     %s\n", cert.NotBefore.Format(time.RFC3339))
	fmt.Fprintf(&b, "Valid Until:    %s (%s)\n",
		cert.NotAfter.Format(time.RFC3339),
		humanizeExpiryDate(cert.NotAfter))
	fmt.Fprintf(&b, "DNS Names:      %s\n", strings.Join(cert.DNSNames, ", "))
	fmt.Fprintf(&b, "Sig Algorithm:  %s\n", cert.SignatureAlgorithm)
	// ... key info, fingerprint, etc.
	return b.String()
}

func humanizeExpiryDate(notAfter time.Time) string {
	d := time.Until(notAfter)
	days := int(d.Hours() / 24)
	if days < 0 {
		return fmt.Sprintf("EXPIRED %d days ago", -days)
	}
	if days == 0 {
		return "expires today"
	}
	return fmt.Sprintf("in %d days", days)
}
