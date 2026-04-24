package crypto

import (
	"crypto/cipher"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20"
)

const chacha20BlockSize = 64

func NewChaCha20AlignedStream(key, nonce []byte, absoluteOffset uint64) (cipher.Stream, error) {
	if len(key) != chacha20.KeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(key))
	}
	if len(nonce) != chacha20.NonceSize {
		return nil, fmt.Errorf("invalid nonce size: %d", len(nonce))
	}

	stream, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return nil, fmt.Errorf("new chacha20 cipher: %w", err)
	}

	blockIndex := absoluteOffset / chacha20BlockSize
	blockOffset := absoluteOffset % chacha20BlockSize
	if blockIndex > uint64(^uint32(0)) {
		return nil, fmt.Errorf("offset too large for chacha20 counter")
	}

	stream.SetCounter(uint32(blockIndex))
	if blockOffset > 0 {
		discard := make([]byte, blockOffset)
		stream.XORKeyStream(discard, discard)
	}

	return stream, nil
}

func NewChaCha20ReaderAtOffset(key, nonce []byte, absoluteOffset uint64, upstream io.Reader) (io.Reader, error) {
	stream, err := NewChaCha20AlignedStream(key, nonce, absoluteOffset)
	if err != nil {
		return nil, err
	}
	return &cipher.StreamReader{S: stream, R: upstream}, nil
}
