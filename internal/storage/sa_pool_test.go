package storage

import (
	"errors"
	"testing"
	"time"
)

func TestSAPoolWeightedRoundRobin(t *testing.T) {
	pool, err := NewSAPool([]SAEntry{
		{ID: "a", Weight: 2, MaxQPS: 100},
		{ID: "b", Weight: 1, MaxQPS: 100},
	})
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}

	got := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		sa, err := pool.Acquire()
		if err != nil {
			t.Fatalf("acquire: %v", err)
		}
		got = append(got, sa.ID)
	}

	countA := 0
	countB := 0
	for _, id := range got {
		if id == "a" {
			countA++
		}
		if id == "b" {
			countB++
		}
	}
	if countA != 4 || countB != 2 {
		t.Fatalf("unexpected distribution a=%d b=%d", countA, countB)
	}
}

func TestSAPoolCircuitBreaker(t *testing.T) {
	pool, err := NewSAPool([]SAEntry{{ID: "a", Weight: 1, MaxQPS: 10}})
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}

	now := time.Now()
	pool.clock = func() time.Time { return now }

	pool.ReportResult("a", errors.New("x"))
	pool.ReportResult("a", errors.New("x"))
	pool.ReportResult("a", errors.New("x"))

	_, err = pool.Acquire()
	if !errors.Is(err, ErrNoSAToken) {
		t.Fatalf("want ErrNoSAToken, got %v", err)
	}

	now = now.Add(3 * time.Second)
	_, err = pool.Acquire()
	if err != nil {
		t.Fatalf("want reopen token, got %v", err)
	}
}
