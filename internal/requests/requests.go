package requests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

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
	X509Info   X509CertificateValidity
	Err        error
}

type X509CertificateValidity struct {
	IsValid         bool
	DaysUntilExpiry int
	Err             error
}

func CreateClient(globalTimeout time.Duration) *Client {
	return &Client{
		client: &http.Client{
			Timeout: globalTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false, // false enabled TLS, which is recommended
				},
			},
		},
	}
}

type Client struct {
	client *http.Client
}

// Public Method of the Client struct to get a simple http response with status code
func (c *Client) QueryHTTP(ctx context.Context, t *config.Target) Result {
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

// Public Method of the Client struct to send a simple tcp connectivity probe
func (c *Client) QueryTCP(ctx context.Context, t *config.Target) Result {
	start := time.Now()

	dialer := &net.Dialer{
		Timeout:   t.Timeout, // per-target timeout from your YAML
		KeepAlive: -1,        // disable keepalive — we close immediately anyway
	}

	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	conn, err := dialer.DialContext(ctx, "tcp", t.Host)
	if err != nil {
		errLatency := time.Since(start)
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Latency: errLatency, Err: err}
	}
	defer conn.Close()
	connLatency := time.Since(start)

	return Result{
		Name:       t.Name,
		Host:       t.Host,
		Type:       t.Type,
		StatusCode: 1,
		Latency:    connLatency,
	}
}

// Public Method of the Client struct to send a simple tcp connectivity probe
func (c *Client) QueryICMP(ctx context.Context, t *config.Target) Result {
	start := time.Now()

	// create a bi-directional layer 3 datasocket for icmp communication
	// udp4 uses unprivilidged icmp echo commands not requiring special capabilities
	// /proc/sys/net/ipv4/ping_group_range needs to read "0       2147483647" on host
	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Err: err}
	}
	defer conn.Close()

	// raw datasockets that are e.g. firewalled block connections, so we set a timeout
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(t.Timeout)
	}
	_ = conn.SetDeadline(deadline)

	/* Construct ICMP Packet with pre-defined packet structure
	- Type (8 bits): Specifies the message type. Echo Request = 8, Echo Reply = 0.
	- Code (8 bits): Provides additional information. For Echo messages, this is always 0.
	- Checksum (16 bits): Used for error-checking the packet.
	- Identifier (16 bits): Helps match requests with replies.
	- Sequence Number (16 bits): Tracks the order of packets.
	- Data (Variable length): Contains the payload, e.g: timestamps.
	*/
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("fiscalismia-monitoring"),
		},
	}
	// Marshaling serializes the actual method into bytes for packet transmission
	wb, err := msg.Marshal(nil)
	if err != nil {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Err: err}
	}

	// the unprivileged ICMP mode used UDP4 socket-address types
	dst := &net.UDPAddr{IP: net.ParseIP(t.Host)}
	// sends the byte encoded icmp packet to the target host
	if _, err := conn.WriteTo(wb, dst); err != nil {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Err: err}
	}

	// allocates a 1500 byte read buffer (Ethernet Maximum Transmission Unit (MTU))
	rb := make([]byte, 1500)
	n, addr, err := conn.ReadFrom(rb)
	latency := time.Since(start)
	if err != nil {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Latency: latency, Err: err}
	}
	if !strings.Contains(addr.String(), t.Host) {
		slog.Warn("Source Address not contained in icmp response.")
	}

	// parses ICMP payload, 1 is IANA's protocol number for ICMP
	// rb[:n] is reading the amount of bytes received
	parsed, err := icmp.ParseMessage(1, rb[:n])
	if err != nil {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Latency: latency, Err: err}
	}
	if parsed.Type != ipv4.ICMPTypeEchoReply {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Latency: latency,
			Err: fmt.Errorf("unexpected icmp type: %v", parsed.Type)}
	}

	return Result{
		Name:       t.Name,
		Host:       t.Host,
		Type:       t.Type,
		StatusCode: 1,
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
				// Validate Date Expiration range to be within expected bounds
				if fromDate.Before(time.Now()) && toDate.After(time.Now()) {
					days := daysUntilExpiry(toDate)
					if days < 7 {
						slog.Warn("Certificate expiring soon", "fromDate", fromDate, "toDate", toDate)
					} else {
						slog.Info("Successfully verified host certificate validity", "CN", cert.Subject.CommonName, "URL", t.URL)
					}
					return X509CertificateValidity{
						IsValid:         true,
						DaysUntilExpiry: days,
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
