package registry

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const maxHTTPResponseBytes = 10 << 20 // 10 MB

// Client owns its transport. A nil transport selects the public-address-only
// production transport. Tests may explicitly inject an isolated transport.
type Client struct{ http *http.Client }

func NewClient(transport http.RoundTripper) *Client {
	if transport == nil {
		transport = &ssrfTransport{}
	}
	return &Client{http: &http.Client{Timeout: 30 * time.Second, Transport: transport, CheckRedirect: registryRedirectPolicy}}
}

var defaultClient = NewClient(nil)

// validateScheme applies to initial requests and every redirect.
func validateScheme(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("registry: URL must use HTTPS with a hostname and no credentials")
	}
	return nil
}

func registryRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) > 3 {
		return fmt.Errorf("registry: too many redirects")
	}
	return validateScheme(req.URL.String())
}

func (client *Client) fetchURL(rawURL string) ([]byte, error) {
	if err := validateScheme(rawURL); err != nil {
		return nil, err
	}
	resp, err := client.http.Get(rawURL) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}
	lr := io.LimitReader(resp.Body, maxHTTPResponseBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxHTTPResponseBytes {
		return nil, fmt.Errorf("response from %s exceeds %d byte limit", rawURL, maxHTTPResponseBytes)
	}
	return data, nil
}

// ssrfTransport validates at dial time and connects to that exact IP. TLS
// still verifies the URL hostname; environment proxies are disabled.
type ssrfTransport struct {
	resolver  resolver
	dial      func(context.Context, string, string) (net.Conn, error)
	once      sync.Once
	transport *http.Transport
}

type resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

func (t *ssrfTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := validateScheme(req.URL.String()); err != nil {
		return nil, err
	}
	t.once.Do(func() {
		t.transport = http.DefaultTransport.(*http.Transport).Clone()
		t.transport.Proxy = nil
		t.transport.DialContext = t.dialContext
		t.transport.DialTLSContext = nil
	})
	return t.transport.RoundTrip(req)
}

func (t *ssrfTransport) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("ssrf: invalid destination")
	}
	addrs, err := t.resolverOrDefault().LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("ssrf: resolution failed: %w", err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("ssrf: empty DNS response")
	}
	for _, addr := range addrs {
		if addr.Zone != "" || isPrivateIP(addr.IP) {
			return nil, fmt.Errorf("ssrf: destination is not public")
		}
	}
	dial := t.dial
	if dial == nil {
		dial = (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	}
	var lastErr error
	for _, addr := range addrs {
		conn, err := dial(ctx, network, net.JoinHostPort(addr.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

func (t *ssrfTransport) resolverOrDefault() resolver {
	if t.resolver != nil {
		return t.resolver
	}
	return net.DefaultResolver
}

// privateNets are the CIDR ranges we block for SSRF.
var privateNets []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
		"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4",
		"fe80::/10", "2001:db8::/32", "2001::/32", "2002::/16", "64:ff9b::/96",
		"::/96", "100::/64", "2001:2::/48", "2001:10::/28", "2001:20::/28", "5f00::/16",
		"::1/128",
		"fc00::/7",
	} {
		_, n, _ := net.ParseCIDR(cidr)
		privateNets = append(privateNets, n)
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() {
		return true
	}
	for _, n := range privateNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
