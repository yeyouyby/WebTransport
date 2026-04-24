package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFetchOnceLengthZero(t *testing.T) {
	client := NewGDriveClient(GDriveClientConfig{BaseURL: "https://example.com", FileID: "x"})
	_, _, err := client.fetchOnce(context.Background(), SAEntry{}, 0, 0)
	if err == nil {
		t.Fatal("expected error for zero length")
	}
}

func TestWaitBackoffContextCanceled(t *testing.T) {
	client := NewGDriveClient(GDriveClientConfig{BaseBackoffMs: 10})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.waitBackoff(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context canceled, got %v", err)
	}
}

func TestBackoffIsBounded(t *testing.T) {
	client := NewGDriveClient(GDriveClientConfig{BaseBackoffMs: 10})
	d := client.backoff(8)
	if d > 2*time.Second {
		t.Fatalf("unexpected backoff duration: %v", d)
	}
}
