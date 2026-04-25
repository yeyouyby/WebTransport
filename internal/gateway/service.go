package gateway

import (
	"context"
	"fmt"
	"io"

	"matrix-gateway/internal/crypto"
	"matrix-gateway/internal/prefetch"
	"matrix-gateway/internal/storage"
)

type Service struct {
	saPool       *storage.SAPool
	gdriveClient *storage.GDriveClient
	key          []byte
	nonce        []byte
	cache        *prefetch.Manager
}

const maxReadRangeBytes = 16 * 1024 * 1024

func NewService(saPool *storage.SAPool, gdriveClient *storage.GDriveClient, key, nonce []byte, cache *prefetch.Manager) *Service {
	return &Service{
		saPool:       saPool,
		gdriveClient: gdriveClient,
		key:          key,
		nonce:        nonce,
		cache:        cache,
	}
}

func (s *Service) StreamRange(ctx context.Context, offset uint64, length uint32, w io.Writer) error {
	if payload, ok := s.cache.Get(offset, length); ok {
		_, err := w.Write(payload)
		return err
	}

	body, _, _, err := s.gdriveClient.FetchRange(ctx, s.saPool, offset, length)
	if err != nil {
		return fmt.Errorf("fetch range: %w", err)
	}
	defer body.Close()

	reader, err := crypto.NewChaCha20ReaderAtOffset(s.key, s.nonce, offset, body)
	if err != nil {
		return fmt.Errorf("create crypto reader: %w", err)
	}

	if _, err := io.Copy(w, reader); err != nil {
		return fmt.Errorf("stream copy: %w", err)
	}
	return nil
}

func (s *Service) ReadRange(ctx context.Context, offset uint64, length uint32) ([]byte, error) {
	if length == 0 {
		return nil, fmt.Errorf("length must be greater than 0")
	}
	if length > maxReadRangeBytes {
		return nil, fmt.Errorf("requested length %d exceeds max allowed %d", length, maxReadRangeBytes)
	}

	if payload, ok := s.cache.Get(offset, length); ok {
		return payload, nil
	}

	body, _, _, err := s.gdriveClient.FetchRange(ctx, s.saPool, offset, length)
	if err != nil {
		return nil, fmt.Errorf("fetch range: %w", err)
	}
	defer body.Close()

	reader, err := crypto.NewChaCha20ReaderAtOffset(s.key, s.nonce, offset, body)
	if err != nil {
		return nil, fmt.Errorf("create crypto reader: %w", err)
	}

	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read decrypted payload: %w", err)
	}
	s.cache.Put(offset, length, payload)
	return payload, nil
}
