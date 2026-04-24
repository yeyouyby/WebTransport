package prefetch

import (
	"testing"
	"time"
)

func TestManagerGetPut(t *testing.T) {
	m := NewManager(2, 2*time.Second)
	m.Put(1, 10, []byte("hello"))
	b, ok := m.Get(1, 10)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(b) != "hello" {
		t.Fatalf("unexpected payload: %s", string(b))
	}
}

func TestManagerExpire(t *testing.T) {
	m := NewManager(2, 20*time.Millisecond)
	m.Put(1, 10, []byte("x"))
	time.Sleep(40 * time.Millisecond)
	if _, ok := m.Get(1, 10); ok {
		t.Fatal("expected cache miss after ttl")
	}
}
