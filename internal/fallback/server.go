package fallback

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"matrix-gateway/internal/sharding"
)

type RangeHandler func(ctx context.Context, offset uint64, length uint32, w io.Writer) error

type ServerConfig struct {
	Addr         string
	CertMap      map[string]string
	RangeHandler RangeHandler
	ShardManager *sharding.Manager
}

type Server struct {
	certManager *CertManager
	httpServer  *http.Server
}

func NewServer(cfg ServerConfig) (*Server, error) {
	cm := NewCertManager()
	for host, pair := range cfg.CertMap {
		parts := strings.Split(pair, ";")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid cert pair for host %s", host)
		}
		if err := cm.LoadExact(host, parts[0], parts[1]); err != nil {
			return nil, err
		}
	}

	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return cm.GetExactCert(info.ServerName)
		},
	}

	h := http.NewServeMux()
	h.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h.HandleFunc("/fallback", func(w http.ResponseWriter, r *http.Request) {
		handleFallback(w, r, cfg.RangeHandler)
	})
	h.HandleFunc("/api/shards", func(w http.ResponseWriter, r *http.Request) {
		handleShardConfig(w, r, cfg.ShardManager)
	})
	h.HandleFunc("/shard-client.js", func(w http.ResponseWriter, r *http.Request) {
		handleShardClientScript(w, r)
	})

	s := &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		TLSConfig:         tlsCfg,
	}

	return &Server{certManager: cm, httpServer: s}, nil
}

func (s *Server) Start() error {
	if s.httpServer.TLSConfig == nil || s.httpServer.TLSConfig.GetCertificate == nil {
		return errors.New("tls get certificate must be configured")
	}
	return s.httpServer.ListenAndServeTLS("", "")
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func handleFallback(w http.ResponseWriter, r *http.Request, handler RangeHandler) {
	offset, length, ok := parseRangeHeader(r.Header.Get("Range"))
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid range"))
		return
	}
	if handler != nil {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusPartialContent)
		if err := handler(r.Context(), offset, length, w); err != nil {
			return
		}
		return
	}
	resp := fmt.Sprintf("offset=%d length=%d", offset, length)
	w.WriteHeader(http.StatusPartialContent)
	_, _ = w.Write([]byte(resp))
}

func handleShardConfig(w http.ResponseWriter, r *http.Request, manager *sharding.Manager) {
	if manager == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "sharding not configured"})
		return
	}
	cfg := manager.Config()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"scheme":      cfg.DefaultScheme,
		"imageShards": cfg.ImageHosts,
		"videoShards": cfg.VideoHosts,
		"audioShards": cfg.AudioHosts,
		"capabilities": map[string]any{
			"imageFormats": []string{"jpg", "jpeg", "png", "webp", "avif", "gif"},
			"videoFormats": []string{"mp4", "m4v", "webm", "mkv", "mov", "ts", "m3u8"},
			"audioFormats": []string{"mp3", "aac", "m4a", "flac", "ogg", "opus", "wav"},
		},
	})
}

func handleShardClientScript(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(shardClientScript)
}

func parseRangeHeader(header string) (uint64, uint32, bool) {
	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "bytes=") {
		return 0, 0, false
	}
	pair := strings.TrimPrefix(header, "bytes=")
	parts := strings.Split(pair, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	start, errS := strconv.ParseUint(parts[0], 10, 64)
	end, errE := strconv.ParseUint(parts[1], 10, 64)
	if errS != nil || errE != nil || end < start {
		return 0, 0, false
	}
	length := end - start + 1
	if length > uint64(^uint32(0)) {
		return 0, 0, false
	}
	return start, uint32(length), true
}
