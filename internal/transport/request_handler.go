package transport

import (
	"context"
	"fmt"
	"time"

	"matrix-gateway/internal/protocol"
)

type RangeReader interface {
	ReadRange(ctx context.Context, offset uint64, length uint32) ([]byte, error)
}

func BuildReliableRequestHandler(reader RangeReader, codec protocol.Codec, secret []byte, ttl time.Duration) SessionHandler {
	return func(ctx context.Context, session Session) {
		for {
			frame, err := session.ReceiveReliable(ctx)
			if err != nil {
				_ = session.Close()
				return
			}
			if len(frame) < protocol.HeaderSize {
				_ = session.Close()
				return
			}

			reqHeader, err := codec.Decode(frame[:protocol.HeaderSize], secret, time.Now(), ttl)
			if err != nil {
				_ = session.Close()
				return
			}

			payload, err := reader.ReadRange(ctx, reqHeader.Offset, reqHeader.Length)
			if err != nil {
				_ = session.Close()
				return
			}

			respHeaderRaw, err := codec.Encode(protocol.Header{
				Timestamp: uint32(time.Now().Unix()),
				RequestID: reqHeader.RequestID,
				Offset:    reqHeader.Offset,
				Length:    uint32(len(payload)),
			}, secret)
			if err != nil {
				_ = session.Close()
				return
			}

			resp := make([]byte, 0, protocol.HeaderSize+len(payload))
			resp = append(resp, respHeaderRaw[:]...)
			resp = append(resp, payload...)
			if err := session.SendReliable(ctx, resp); err != nil {
				_ = session.Close()
				return
			}
		}
	}
}

func ParseResponseFrame(codec protocol.Codec, secret []byte, ttl time.Duration, frame []byte, now time.Time) (protocol.Header, []byte, error) {
	if len(frame) < protocol.HeaderSize {
		return protocol.Header{}, nil, fmt.Errorf("invalid response frame length: %d", len(frame))
	}
	h, err := codec.Decode(frame[:protocol.HeaderSize], secret, now, ttl)
	if err != nil {
		return protocol.Header{}, nil, err
	}
	return h, frame[protocol.HeaderSize:], nil
}
