package fallback

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"matrix-gateway/internal/protocol"
)

type delayedMediaWriter struct {
	w           http.ResponseWriter
	wroteHeader bool
	headerBuf   []byte
}

func (d *delayedMediaWriter) Header() http.Header {
	return d.w.Header()
}

func (d *delayedMediaWriter) WriteHeader(statusCode int) {
	if d.wroteHeader {
		return
	}
	d.wroteHeader = true
	d.w.WriteHeader(statusCode)
}

func (d *delayedMediaWriter) Write(p []byte) (int, error) {
	if !d.wroteHeader {
		d.w.Header().Set("Content-Type", "application/octet-stream")
		d.WriteHeader(http.StatusOK)
		if len(d.headerBuf) > 0 {
			_, err := d.w.Write(d.headerBuf)
			if err != nil {
				return 0, err
			}
		}
	}
	return d.w.Write(p)
}

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

	if handler == nil {
		http.Error(w, "Internal Server Error: nil RangeHandler", http.StatusInternalServerError)
		return
	}

	respHeaderRaw, err := codec.Encode(protocol.Header{
		Timestamp: uint32(time.Now().Unix()),
		RequestID: reqHeader.RequestID,
		Offset:    reqHeader.Offset,
		Length:    reqHeader.Length,
	}, secret)
	if err != nil {
		http.Error(w, "Failed to encode response header", http.StatusInternalServerError)
		return
	}

	// We use delayedMediaWriter to avoid sending 200 OK before we know the range fetch succeeds
	dw := &delayedMediaWriter{w: w, headerBuf: respHeaderRaw[:]}

	if err := handler(r.Context(), reqHeader.Offset, reqHeader.Length, dw); err != nil {
		if !dw.wroteHeader {
			http.Error(w, "upstream fetch failed", http.StatusBadGateway)
		}
		return
	}

	// If the handler didn't write anything (e.g. empty file) but succeeded, flush the headers
	if !dw.wroteHeader {
		dw.w.Header().Set("Content-Type", "application/octet-stream")
		dw.w.WriteHeader(http.StatusOK)
		_, _ = dw.w.Write(dw.headerBuf)
	}
}
