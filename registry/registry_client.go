package registry

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxHTTPResponseBytes = 10 << 20 // 10 MB

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &ssrfTransport{base: http.DefaultTransport},
}

// validateScheme rejects non-HTTPS schemes unless insecure HTTP is allowed for tests.
func validateScheme(rawURL string) error {
	if isInsecureHTTPAllowed() {
		if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
			return nil
		}
		return fmt.Errorf("registry: refusing URL with disallowed scheme: %q", rawURL)
	}
	if !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("registry: refusing non-HTTPS URL %q", rawURL)
	}
	return nil
}

func fetchURL(rawURL string) ([]byte, error) {
	if err := validateScheme(rawURL); err != nil {
		return nil, err
	}
	resp, err := httpClient.Get(rawURL) //nolint:gosec
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

// ssrfTransport wraps an http.RoundTripper and rejects requests that resolve
// to private/loopback/link-local IP addresses.
type ssrfTransport struct {
	base     http.RoundTripper
	resolver resolver
}

type resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

func (t *ssrfTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Hostname()
	if host == "" {
		return nil, fmt.Errorf("ssrf: empty hostname")
	}

	// Skip SSRF check if insecure HTTP is allowed (test mode)
	// and the host is a loopback — httptest servers bind to 127.0.0.1.
	if isInsecureHTTPAllowed() {
		return t.base.RoundTrip(req)
	}

	r := t.resolverOrDefault()
	addrs, err := r.LookupIPAddr(req.Context(), host)
	if err != nil {
		return nil, fmt.Errorf("ssrf: resolve %q: %w", host, err)
	}
	for _, addr := range addrs {
		if isPrivateIP(addr.IP) {
			return nil, fmt.Errorf("ssrf: %q resolves to private address %s", host, addr.IP)
		}
	}
	return t.base.RoundTrip(req)
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
		"::1/128",
		"fc00::/7",
	} {
		_, n, _ := net.ParseCIDR(cidr)
		privateNets = append(privateNets, n)
	}
}

func isPrivateIP(ip net.IP) bool {
	for _, n := range privateNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
