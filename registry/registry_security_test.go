package registry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"
)

type fixedResolver []net.IPAddr

func (r fixedResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) { return r, nil }

func TestRegistryRejectsNonPublicAddresses(t *testing.T) {
	for _, address := range []string{"0.0.0.0", "127.0.0.1", "10.0.0.1", "100.64.0.1", "169.254.169.254", "192.0.0.1", "198.18.0.1", "224.0.0.1", "::", "::1", "fe80::1", "fc00::1", "ff02::1", "::ffff:127.0.0.1"} {
		if !isPrivateIP(net.ParseIP(address)) {
			t.Errorf("accepted non-public address %s", address)
		}
	}
}

func TestRegistryDialUsesOnlyValidatedResolution(t *testing.T) {
	for _, addresses := range []fixedResolver{nil, {{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("127.0.0.1")}}} {
		transport := &ssrfTransport{resolver: addresses}
		if _, err := transport.dialContext(context.Background(), "tcp", "registry.example:443"); err == nil {
			t.Fatal("expected empty or mixed-public DNS refusal")
		}
	}
	var dialed string
	transport := &ssrfTransport{resolver: fixedResolver{{IP: net.ParseIP("8.8.8.8")}}, dial: func(_ context.Context, _, address string) (net.Conn, error) {
		dialed = address
		return nil, errors.New("test dial stopped")
	}}
	_, _ = transport.dialContext(context.Background(), "tcp", "registry.example:443")
	if dialed != "8.8.8.8:443" {
		t.Fatalf("dial did not use the validated IP: %q", dialed)
	}
}

func TestRegistryRedirectPolicy(t *testing.T) {
	for _, raw := range []string{"http://registry.example/file", "https://user:password@registry.example/file", "file:///etc/passwd", "https:///file"} {
		if err := validateScheme(raw); err == nil {
			t.Errorf("accepted unsafe URL %q", raw)
		}
	}
	address, err := url.Parse("https://registry.example/file")
	if err != nil {
		t.Fatal(err)
	}
	request := &http.Request{URL: address}
	if err := registryRedirectPolicy(request, make([]*http.Request, 3)); err != nil {
		t.Fatal(err)
	}
	if err := registryRedirectPolicy(request, make([]*http.Request, 4)); err == nil {
		t.Fatal("accepted fourth redirect")
	}
}
