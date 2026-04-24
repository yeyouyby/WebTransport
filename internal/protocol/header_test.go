package protocol

import (
	"errors"
	"testing"
	"time"
)

func TestCodecEncodeDecode(t *testing.T) {
	codec := Codec{}
	secret := []byte("test-secret")
	now := time.Now()
	h := Header{
		Timestamp: uint32(now.Unix()),
		RequestID: 123,
		Offset:    456,
		Length:    2048,
	}

	raw, err := codec.Encode(h, secret)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := codec.Decode(raw[:], secret, now, 10*time.Second)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if got.RequestID != h.RequestID || got.Offset != h.Offset || got.Length != h.Length {
		t.Fatalf("decoded mismatch: %+v", got)
	}
}

func TestCodecDecodeExpired(t *testing.T) {
	codec := Codec{}
	secret := []byte("test-secret")
	h := Header{Timestamp: uint32(time.Now().Add(-30 * time.Second).Unix())}
	raw, _ := codec.Encode(h, secret)

	_, err := codec.Decode(raw[:], secret, time.Now(), 5*time.Second)
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("want ErrExpired, got %v", err)
	}
}
