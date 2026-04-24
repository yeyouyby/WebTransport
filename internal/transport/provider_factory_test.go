package transport

import "testing"

func TestNewProvider(t *testing.T) {
	p, err := NewProvider(ProviderConfig{Mode: ProviderModeStub})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	if p == nil {
		t.Fatal("provider is nil")
	}
}

func TestEndpointWithClientID(t *testing.T) {
	base := "https://example.com/wt?x=1"
	got := endpointWithClientID(base, "client-1")
	want := "https://example.com/wt?client_id=client-1&x=1"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestNewWebTransportProviderRequireHTTPS(t *testing.T) {
	_, err := NewWebTransportProvider(WebTransportConfig{EndpointURL: "http://example.com/wt"})
	if err == nil {
		t.Fatal("expected https validation error")
	}
}
