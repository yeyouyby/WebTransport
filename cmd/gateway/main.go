package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"matrix-gateway/internal/config"
	"matrix-gateway/internal/fallback"
	"matrix-gateway/internal/gateway"
	"matrix-gateway/internal/prefetch"
	"matrix-gateway/internal/protocol"
	"matrix-gateway/internal/sharding"
	"matrix-gateway/internal/storage"
	"matrix-gateway/internal/transport"
)

func main() {
	cfg := config.LoadFromEnv()
	if len(cfg.Fallback.CertMap) == 0 {
		log.Fatal("fallback cert map is empty, set TLS_CERT_MAP with exact SNI certificates")
	}
	if transport.CCPolicy(cfg.Transport.CCPolicy) == transport.CCPolicyBrutal {
		log.Println("brutal policy enabled; forked quic-go is required and already expected via go.mod replace")
	}

	key, nonce, err := loadCryptoMaterial(cfg.Crypto.KeyHex, cfg.Crypto.NonceHex)
	if err != nil {
		log.Fatalf("load crypto material: %v", err)
	}

	saPool, err := storage.NewSAPool(cfg.Storage.SAEntries)
	if err != nil {
		log.Fatalf("init sa pool: %v", err)
	}

	gdriveClient := storage.NewGDriveClient(storage.GDriveClientConfig{
		BaseURL:       cfg.Storage.GDriveBaseURL,
		FileID:        cfg.Storage.GDriveFileID,
		MaxRetries:    cfg.Storage.MaxRetries,
		BaseBackoffMs: cfg.Storage.BaseBackoffMs,
	})

	codec := protocol.Codec{}
	prefetchCache := prefetch.NewManager(cfg.Prefetch.MaxEntries, time.Duration(cfg.Prefetch.TTLSeconds)*time.Second)
	gatewayService := gateway.NewService(saPool, gdriveClient, key, nonce, prefetchCache)

	shardManager, err := sharding.NewManager(sharding.Config{
		ImageHosts:    cfg.Sharding.ImageHosts,
		VideoHosts:    cfg.Sharding.VideoHosts,
		AudioHosts:    cfg.Sharding.AudioHosts,
		DefaultScheme: cfg.Sharding.DefaultScheme,
	})
	if err != nil {
		log.Fatalf("init shard manager: %v", err)
	}

	provider, err := transport.NewProvider(transport.ProviderConfig{
		Mode:            transport.ProviderMode(cfg.Transport.Provider),
		CCPolicy:        transport.CCPolicy(cfg.Transport.CCPolicy),
		BrutalTargetBPS: cfg.Transport.BrutalTargetBPS,
		BrutalRTTMs:     cfg.Transport.BrutalRTTMs,
		EndpointURL:     cfg.Transport.EndpointURL,
		InsecureTLS:     cfg.Transport.InsecureTLS,
	})
	if err != nil {
		log.Fatalf("init transport provider: %v", err)
	}
	_ = provider

	h2Server, err := fallback.NewServer(fallback.ServerConfig{
		Addr:         cfg.Fallback.Addr,
		CertMap:      cfg.Fallback.CertMap,
		ShardManager: shardManager,
		RangeHandler: func(ctx context.Context, offset uint64, length uint32, w io.Writer) error {
			return gatewayService.StreamRange(ctx, offset, length, w)
		},
	})
	if err != nil {
		log.Fatalf("init fallback server: %v", err)
	}

	var wtServer *transport.WebTransportServer
	if strings.EqualFold(cfg.Transport.Provider, string(transport.ProviderModeWebTransport)) || cfg.Transport.ServerEnabled {
		if cfg.Transport.ServerCertFile == "" || cfg.Transport.ServerKeyFile == "" {
			log.Fatal("webtransport server enabled but cert or key file is empty")
		}
		wtServer, err = transport.NewWebTransportServer(transport.WebTransportServerConfig{
			Addr:      cfg.Transport.ServerAddr,
			Path:      cfg.Transport.ServerPath,
			CertFile:  cfg.Transport.ServerCertFile,
			KeyFile:   cfg.Transport.ServerKeyFile,
			CCPolicy:  transport.CCPolicy(cfg.Transport.CCPolicy),
			TargetBPS: cfg.Transport.BrutalTargetBPS,
			RTTMs:     cfg.Transport.BrutalRTTMs,
			Handler:   transport.BuildReliableRequestHandler(gatewayService, codec, []byte(cfg.Protocol.TokenSecret), cfg.Protocol.HeaderTTL),
			HTTPHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/healthz" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("ok"))
					return
				}
				http.NotFound(w, r)
			}),
		})
		if err != nil {
			log.Fatalf("init webtransport server: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	go func() {
		errCh <- h2Server.Start()
	}()
	if wtServer != nil {
		go func() {
			errCh <- wtServer.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server exited with error: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Fallback.ShutdownTimeout)
	defer cancel()
	if err := h2Server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown fallback server: %v", err)
	}
	if wtServer != nil {
		if err := wtServer.Close(); err != nil {
			log.Printf("shutdown webtransport server: %v", err)
		}
	}

	os.Exit(0)
}

func loadCryptoMaterial(keyHex, nonceHex string) ([]byte, []byte, error) {
	if keyHex == "" || nonceHex == "" {
		return nil, nil, fmt.Errorf("CRYPTO_KEY_HEX and CRYPTO_NONCE_HEX are required")
	}
	key, err := hex.DecodeString(strings.TrimSpace(keyHex))
	if err != nil {
		return nil, nil, fmt.Errorf("decode key hex: %w", err)
	}
	nonce, err := hex.DecodeString(strings.TrimSpace(nonceHex))
	if err != nil {
		return nil, nil, fmt.Errorf("decode nonce hex: %w", err)
	}
	return key, nonce, nil
}
