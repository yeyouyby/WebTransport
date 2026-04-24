package transport

import (
	"context"
	"sync"
	"testing"
	"time"

	"matrix-gateway/internal/protocol"
)

type mockRangeReader struct {
	data []byte
}

func (m *mockRangeReader) ReadRange(_ context.Context, _ uint64, _ uint32) ([]byte, error) {
	out := make([]byte, len(m.data))
	copy(out, m.data)
	return out, nil
}

type mockSession struct {
	mu       sync.Mutex
	closed   bool
	incoming chan []byte
	outgoing chan []byte
}

func newMockSession() *mockSession {
	return &mockSession{incoming: make(chan []byte, 2), outgoing: make(chan []byte, 2)}
}

func (m *mockSession) SendReliable(_ context.Context, payload []byte) error {
	m.outgoing <- payload
	return nil
}

func (m *mockSession) ReceiveReliable(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-m.incoming:
		return data, nil
	}
}

func (m *mockSession) SendUnreliable(_ context.Context, _ []byte) error { return nil }
func (m *mockSession) ReceiveUnreliable(_ context.Context) ([]byte, error) {
	return nil, context.Canceled
}
func (m *mockSession) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func TestBuildReliableRequestHandler(t *testing.T) {
	codec := protocol.Codec{}
	secret := []byte("test-secret")
	reader := &mockRangeReader{data: []byte("payload")}
	h := BuildReliableRequestHandler(reader, codec, secret, 10*time.Second)

	reqRaw, err := codec.Encode(protocol.Header{
		Timestamp: uint32(time.Now().Unix()),
		RequestID: 7,
		Offset:    10,
		Length:    20,
	}, secret)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}

	sess := newMockSession()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h(ctx, sess)

	sess.incoming <- reqRaw[:]

	select {
	case out := <-sess.outgoing:
		hdr, payload, err := ParseResponseFrame(codec, secret, 10*time.Second, out, time.Now())
		if err != nil {
			t.Fatalf("parse response: %v", err)
		}
		if hdr.RequestID != 7 {
			t.Fatalf("unexpected request id: %d", hdr.RequestID)
		}
		if string(payload) != "payload" {
			t.Fatalf("unexpected payload: %s", string(payload))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}
