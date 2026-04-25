package fallback

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"matrix-gateway/internal/protocol"
)

func handleMediaFallback(w http.ResponseWriter, r *http.Request, handler RangeHandler, codec protocol.Codec, secret []byte, ttl time.Duration) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, protocol.HeaderSize))
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(bodyBytes) < protocol.HeaderSize {
		http.Error(w, "Invalid header length", http.StatusBadRequest)
		return
	}

	reqHeader, err := codec.Decode(bodyBytes[:protocol.HeaderSize], secret, time.Now(), ttl)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode header: %v", err), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	respHeaderRaw, err := codec.Encode(protocol.Header{
		Timestamp: uint32(time.Now().Unix()),
		RequestID: reqHeader.RequestID,
		Offset:    reqHeader.Offset,
		Length:    reqHeader.Length,
	}, secret)
	if err == nil {
		w.Write(respHeaderRaw[:])
	}

	_ = handler(r.Context(), reqHeader.Offset, reqHeader.Length, w)
}
