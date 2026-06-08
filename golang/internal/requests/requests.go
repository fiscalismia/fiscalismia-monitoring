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
	TYPE_DIVIDER              = "divider_control_seq"
	TYPE_QUERY_DURATION       = "query_duration_control_seq"
)

type Result struct {
	Name       string
	URL        string
	Host       string
	Type       string
	StatusCode int
	Body       string
	Latency    time.Duration
	ExpectFail bool
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

	// Apply Target timeout to Context
	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		return Result{Name: t.Name, URL: t.URL, Type: t.Type, ExpectFail: t.ExpectFail, Err: err}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		errLatency := time.Since(start)
		return Result{Name: t.Name, URL: t.URL, Type: t.Type, Latency: errLatency, ExpectFail: t.ExpectFail, Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, READ_CAPACITY_BYTES))
	if err != nil {
		errLatency := time.Since(start)
		return Result{Name: t.Name, URL: t.URL, Type: t.Type, StatusCode: resp.StatusCode, Latency: errLatency, ExpectFail: t.ExpectFail, Err: err}
	}
	detail := string(body)

	if strings.HasPrefix(detail, "<!doctype html>") {
		// removes uneven and even whitespace with len > 1
		// single whitespaces within html tags are retained
		detail = strings.ReplaceAll(string(body), "   ", "")
		detail = strings.ReplaceAll(string(body), "  ", "")
		detail = strings.ReplaceAll(detail, "\n", "")
	}
	latency := time.Since(start)
	return Result{
		Name:       t.Name,
		URL:        t.URL,
		Type:       t.Type,
		ExpectFail: t.ExpectFail,
		StatusCode: resp.StatusCode,
		Body:       detail,
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
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Latency: errLatency, ExpectFail: t.ExpectFail, Err: err}
	}
	defer conn.Close()
	connLatency := time.Since(start)

	return Result{
		Name:       t.Name,
		Host:       t.Host,
		Type:       t.Type,
		ExpectFail: t.ExpectFail,
		X509Info:   X509CertificateValidity{IsValid: false, DaysUntilExpiry: -1, Err: errors.New("layer 4 TCP Protocol does not speak Layer 7 TLS Certificates")},
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
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, ExpectFail: t.ExpectFail, Err: err}
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
	dataPayload := time.Now().String()
	slog.Debug("dataPayload", "d", dataPayload)
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte(dataPayload),
		},
	}
	// Marshaling serializes the actual method into bytes for packet transmission
	wb, err := msg.Marshal(nil)
	if err != nil {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, ExpectFail: t.ExpectFail, Err: err}
	}

	// the unprivileged ICMP mode used UDP4 socket-address types
	dst := &net.UDPAddr{IP: net.ParseIP(t.Host)}
	// sends the byte encoded icmp packet to the target host
	if _, err := conn.WriteTo(wb, dst); err != nil {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, ExpectFail: t.ExpectFail, Err: err}
	}

	// allocates a 1500 byte read buffer (Ethernet Maximum Transmission Unit (MTU))
	rb := make([]byte, 1500)
	n, addr, err := conn.ReadFrom(rb)
	latency := time.Since(start)
	if err != nil {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Latency: latency, ExpectFail: t.ExpectFail, Err: err}
	}
	if !strings.Contains(addr.String(), t.Host) {
		slog.Warn("Source Address not contained in icmp response.")
	}

	// parses ICMP payload, 1 is IANA's protocol number for ICMP
	// rb[:n] is reading the amount of bytes received
	parsed, err := icmp.ParseMessage(1, rb[:n])
	if err != nil {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Latency: latency, ExpectFail: t.ExpectFail, Err: err}
	}
	if parsed.Type != ipv4.ICMPTypeEchoReply {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Latency: latency,
			Err: fmt.Errorf("unexpected icmp type: %v", parsed.Type)}
	}
	// extracts the received payload
	echo, ok := parsed.Body.(*icmp.Echo)
	if !ok {
		return Result{Name: t.Name, Host: t.Host, Type: t.Type, Latency: latency,
			Err: fmt.Errorf("unexpected ICMP body type: %T", parsed.Body)}
	}
	// should contain exactly the dataPayload sent
	responseBody := string(echo.Data)

	return Result{
		Name:       t.Name,
		Host:       t.Host,
		Type:       t.Type,
		ExpectFail: t.ExpectFail,
		X509Info:   X509CertificateValidity{IsValid: false, DaysUntilExpiry: -1, Err: errors.New("ICMP Protocol does not speak TLS Certificates")},
		StatusCode: 1,
		Latency:    latency,
		Body:       responseBody,
	}
}

// Public Method of the Client struct to query its tls certs for validity dates checking expiry
func (c *Client) VerifyTLSCertificate(ctx context.Context, t *config.Target, rd string) X509CertificateValidity {
	// sanity check for target URL
	if !strings.HasPrefix(t.URL, "https://") {
		return X509CertificateValidity{IsValid: false, Err: errors.New("URL does not contain https:// protocol")}
	}

	tlsUrl, err := cleanTlsURL(t.URL, rd)
	if err != nil {
		return X509CertificateValidity{IsValid: false, DaysUntilExpiry: -1, Err: err}
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}

	// Bound dial + handshake by the per-target timeout, and inherit cancellation
	// from the parent ctx so SIGINT/SIGTERM aborts an in-flight connect.
	dialCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{},
		Config:    tlsConfig,
	}
	conn, err := dialer.DialContext(dialCtx, "tcp", tlsUrl)
	if err != nil {
		return X509CertificateValidity{IsValid: false, DaysUntilExpiry: -1, Err: err}
	}
	defer conn.Close()

	tlsConn := conn.(*tls.Conn)
	certs := tlsConn.ConnectionState().PeerCertificates

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
		Err:             errors.New("VerifyTLSCertificate completed without resolution"),
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
