package crypto

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

func TestNewChaCha20ReaderAtOffset(t *testing.T) {
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	plain := make([]byte, 256*1024)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand key: %v", err)
	}
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("rand nonce: %v", err)
	}
	if _, err := rand.Read(plain); err != nil {
		t.Fatalf("rand plain: %v", err)
	}

	encryptedReader, err := NewChaCha20ReaderAtOffset(key, nonce, 0, bytes.NewReader(plain))
	if err != nil {
		t.Fatalf("create encrypt reader: %v", err)
	}
	encrypted, err := io.ReadAll(encryptedReader)
	if err != nil {
		t.Fatalf("read encrypted: %v", err)
	}

	cases := []struct {
		offset uint64
		length int
	}{
		{0, 1024},
		{1, 4096},
		{63, 4096},
		{64, 4096},
		{513, 16 * 1024},
		{80 * 1024, 12 * 1024},
	}

	for _, tc := range cases {
		t.Run("offset", func(t *testing.T) {
			start := int(tc.offset)
			end := start + tc.length
			if end > len(encrypted) {
				end = len(encrypted)
			}

			decReader, err := NewChaCha20ReaderAtOffset(key, nonce, tc.offset, bytes.NewReader(encrypted[start:end]))
			if err != nil {
				t.Fatalf("create decrypt reader: %v", err)
			}
			dec, err := io.ReadAll(decReader)
			if err != nil {
				t.Fatalf("read decrypt: %v", err)
			}

			want := plain[start:end]
			if !bytes.Equal(dec, want) {
				t.Fatalf("decrypt mismatch at offset=%d", tc.offset)
			}
		})
	}
}

func BenchmarkChaCha20StreamAtOffset(b *testing.B) {
	key := bytes.Repeat([]byte{0x11}, 32)
	nonce := bytes.Repeat([]byte{0x22}, 12)
	data := bytes.Repeat([]byte("abcd"), 16*1024)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		reader, err := NewChaCha20ReaderAtOffset(key, nonce, 1024, bytes.NewReader(data))
		if err != nil {
			b.Fatalf("new reader: %v", err)
		}
		if _, err := io.Copy(io.Discard, reader); err != nil {
			b.Fatalf("copy: %v", err)
		}
	}
}
