package transport

import (
	"context"
	"errors"
	"sync"
)

type StubProvider struct {
	policy CCPolicy
	brutal BrutalConfig
}

func NewStubProvider(policy CCPolicy, targetBPS int64, rttMs int) *StubProvider {
	if targetBPS <= 0 {
		targetBPS = 50_000_000
	}
	if rttMs <= 0 {
		rttMs = 120
	}
	return &StubProvider{
		policy: policy,
		brutal: BrutalConfig{TargetBPS: targetBPS, RTTMs: rttMs},
	}
}

func (p *StubProvider) Open(_ context.Context, clientID string) (Session, error) {
	if clientID == "" {
		return nil, errors.New("empty client id")
	}
	return &stubSession{}, nil
}

type stubSession struct {
	mu              sync.Mutex
	closed          bool
	reliableBytes   int64
	unreliableBytes int64
}

func (s *stubSession) SendReliable(_ context.Context, payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("session closed")
	}
	s.reliableBytes += int64(len(payload))
	return nil
}

func (s *stubSession) ReceiveReliable(_ context.Context) ([]byte, error) {
	return nil, errors.New("receive stream is not supported in stub session")
}

func (s *stubSession) SendUnreliable(_ context.Context, payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("session closed")
	}
	s.unreliableBytes += int64(len(payload))
	return nil
}

func (s *stubSession) ReceiveUnreliable(_ context.Context) ([]byte, error) {
	return nil, errors.New("receive datagram is not supported in stub session")
}

func (s *stubSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}
