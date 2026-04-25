package transport

import (
	"context"
	"strings"
	"testing"
)

func TestSendReliableRejectsOversizedPayload(t *testing.T) {
	s := &webTransportSession{}
	payload := make([]byte, maxReliableFrameBytes+1)
	err := s.SendReliable(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for oversized reliable frame")
	}
	if !strings.Contains(err.Error(), "reliable frame too large") {
		t.Fatalf("unexpected error: %v", err)
	}
}
