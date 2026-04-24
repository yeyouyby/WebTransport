package protocol

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	HeaderSize = 32
	Magic      = 0x4D4D4746
)

var (
	ErrInvalidMagic = errors.New("invalid magic")
	ErrExpired      = errors.New("expired timestamp")
	ErrInvalidToken = errors.New("invalid token")
)

type Header struct {
	Timestamp uint32
	RequestID uint64
	Offset    uint64
	Length    uint32
	Token     uint32
}

type Codec struct{}

func (Codec) Encode(h Header, secret []byte) ([HeaderSize]byte, error) {
	var out [HeaderSize]byte
	binary.BigEndian.PutUint32(out[0:4], Magic)
	binary.BigEndian.PutUint32(out[4:8], h.Timestamp)
	binary.BigEndian.PutUint64(out[8:16], h.RequestID)
	binary.BigEndian.PutUint64(out[16:24], h.Offset)
	binary.BigEndian.PutUint32(out[24:28], h.Length)
	binary.BigEndian.PutUint32(out[28:32], signToken(out[0:28], secret))
	return out, nil
}

func (Codec) Decode(raw []byte, secret []byte, now time.Time, ttl time.Duration) (Header, error) {
	if len(raw) < HeaderSize {
		return Header{}, fmt.Errorf("invalid header length: %d", len(raw))
	}
	magic := binary.BigEndian.Uint32(raw[0:4])
	if magic != Magic {
		return Header{}, ErrInvalidMagic
	}

	ts := binary.BigEndian.Uint32(raw[4:8])
	if ttl > 0 {
		requestTime := time.Unix(int64(ts), 0)
		if now.Sub(requestTime) > ttl {
			return Header{}, ErrExpired
		}
	}

	token := binary.BigEndian.Uint32(raw[28:32])
	expect := signToken(raw[0:28], secret)
	if token != expect {
		return Header{}, ErrInvalidToken
	}

	return Header{
		Timestamp: ts,
		RequestID: binary.BigEndian.Uint64(raw[8:16]),
		Offset:    binary.BigEndian.Uint64(raw[16:24]),
		Length:    binary.BigEndian.Uint32(raw[24:28]),
		Token:     token,
	}, nil
}

func signToken(headerWithoutToken []byte, secret []byte) uint32 {
	h := hmac.New(sha256.New, secret)
	_, _ = h.Write(headerWithoutToken)
	sum := h.Sum(nil)
	return binary.BigEndian.Uint32(sum[:4])
}
