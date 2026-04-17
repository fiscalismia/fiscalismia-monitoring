package requests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
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
	IsValid         bool
	DaysUntilExpiry int
	Err             error
}

type Client struct {
	client *http.Client
}

// Public Method of the Client struct to get a simple http response with status code
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

// Public Method of the Client struct to query its tls certs for validity dates checking expiry
func (c *Client) VerifyTLSCertificate(ctx context.Context, t *config.Target, rd string) X509CertificateValidity {
	// sanity check for target URL
	if !strings.HasPrefix(t.URL, "https://") {
		return X509CertificateValidity{IsValid: false, Err: errors.New("URL does not contain https:// protocol.")}
	}

	tlsUrl, err := cleanTlsURL(t.URL, rd)
	if err != nil {
		return X509CertificateValidity{IsValid: false, DaysUntilExpiry: -1, Err: err}
	}

	config := &tls.Config{
		InsecureSkipVerify: false,
	}

	// Establish a TCP connection first
	conn, err := tls.Dial("tcp", tlsUrl, config)
	if err != nil {
		return X509CertificateValidity{IsValid: false, DaysUntilExpiry: -1, Err: err}
	}
	defer conn.Close()

	// Perform SSL/TLS handshake
	err = conn.Handshake()
	if err != nil {
		return X509CertificateValidity{IsValid: false, DaysUntilExpiry: -1, Err: err}
	}

	// Retrieve the peer certificates
	certs := conn.ConnectionState().PeerCertificates

	// Iterate and validate each certificate in the chain if tls verify flag is set in conf
	for _, cert := range certs {
		if cert.Subject.CommonName != "" && len(cert.DNSNames) > 0 {
			if strings.Contains(tlsUrl, cert.Subject.CommonName) {
				slog.Default().LogAttrs(ctx, slog.LevelDebug, "====> Detailed x509 certificate <====", x509CertAttrs(cert)...)
				fromDate := cert.NotBefore
				toDate := cert.NotAfter
				if fromDate.Before(time.Now()) && toDate.After(time.Now()) {
					slog.Debug("Successfully verified host certificate validity", "URL", t.URL, "CN", cert.Subject.CommonName)
					return X509CertificateValidity{
						IsValid:         true,
						DaysUntilExpiry: daysUntilExpiry(toDate),
					}
				} else {
					slog.Warn("Certificate validity time out of range.", "fromDate", fromDate, "toDate", toDate)
					break
				}

			}
		}
	}

	return X509CertificateValidity{
		IsValid:         false,
		DaysUntilExpiry: -1,
		Err:             errors.New("VerifyTLSCertificate completed without resolution."),
	}
}

func CreateClient(globalTimeout time.Duration) *Client {
	return &Client{
		client: &http.Client{
			Timeout: globalTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
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

// x509CertAttrs returns a grouped slog attribute containing TLS certificate fields.
func x509CertAttrs(cert *x509.Certificate) []slog.Attr {
	return []slog.Attr{
		slog.String("subject", cert.Subject.CommonName),
		slog.String("issuer", cert.Issuer.CommonName),
		slog.String("issuer_url", cert.IssuingCertificateURL[0]),
		slog.String("serial", cert.SerialNumber.Text(16)),
		slog.String("valid_from", cert.NotBefore.Format(time.RFC3339)),
		slog.String("valid_until", cert.NotAfter.Format(time.RFC3339)),
		slog.Int("expires_in_days", daysUntilExpiry(cert.NotAfter)),
		slog.Any("dns_names", cert.DNSNames),
		slog.String("sig_algorithm", cert.SignatureAlgorithm.String()),
	}
}

func daysUntilExpiry(t time.Time) int {
	return int(time.Until(t).Hours() / 24)
}
