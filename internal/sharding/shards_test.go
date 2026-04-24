package sharding

import "testing"

func TestPickHostStable(t *testing.T) {
	m, err := NewManager(Config{
		ImageHosts: []string{"s1.api.com", "s2.api.com"},
		VideoHosts: []string{"v1.api.com", "v2.api.com", "v3.api.com"},
		AudioHosts: []string{"a1.api.com", "a2.api.com"},
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	h1, err := m.PickHost(ResourceVideo, "file-123:chunk-9")
	if err != nil {
		t.Fatalf("pick host: %v", err)
	}
	h2, err := m.PickHost(ResourceVideo, "file-123:chunk-9")
	if err != nil {
		t.Fatalf("pick host: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("host selection should be stable: %s vs %s", h1, h2)
	}
}

func TestPickHostByType(t *testing.T) {
	m, err := NewManager(Config{
		ImageHosts: []string{"s1.api.com"},
		VideoHosts: []string{"v1.api.com"},
		AudioHosts: []string{"a1.api.com"},
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	h, err := m.PickHost(ResourceImage, "cover")
	if err != nil {
		t.Fatalf("pick image host: %v", err)
	}
	if h != "s1.api.com" {
		t.Fatalf("unexpected image host: %s", h)
	}

	_, err = m.PickHost("other", "x")
	if err == nil {
		t.Fatal("expected error for unknown resource type")
	}

	a, err := m.PickHost(ResourceAudio, "track-1")
	if err != nil {
		t.Fatalf("pick audio host: %v", err)
	}
	if a != "a1.api.com" {
		t.Fatalf("unexpected audio host: %s", a)
	}
}
