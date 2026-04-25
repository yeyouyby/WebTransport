package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	quic "github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	webtransport "github.com/quic-go/webtransport-go"
)

type SessionHandler func(ctx context.Context, session Session)

type WebTransportServerConfig struct {
	Addr        string
	CertFile    string
	KeyFile     string
	CCPolicy    CCPolicy
	TargetBPS   int64
	RTTMs       int
	Path        string
	Handler     SessionHandler
	TLSConfig   *tls.Config
	HTTPHandler http.Handler
}

type WebTransportServer struct {
	cfg WebTransportServerConfig
	wt  *webtransport.Server
	mu  sync.Mutex
}

func NewWebTransportServer(cfg WebTransportServerConfig) (*WebTransportServer, error) {
	if cfg.Path == "" {
		cfg.Path = "/wt"
	}
	if cfg.Addr == "" {
		cfg.Addr = ":443"
	}
	if cfg.TargetBPS <= 0 {
		cfg.TargetBPS = 50_000_000
	}
	if cfg.RTTMs <= 0 {
		cfg.RTTMs = 120
	}
	if cfg.TLSConfig == nil {
		cfg.TLSConfig = &tls.Config{}
	}
	if !containsString(cfg.TLSConfig.NextProtos, "h3") {
		cfg.TLSConfig.NextProtos = append(cfg.TLSConfig.NextProtos, "h3")
	}

	quicCfg := &quic.Config{EnableDatagrams: true, EnableStreamResetPartialDelivery: true}
	if cfg.CCPolicy == CCPolicyBrutal {
		quicCfg.InitialConnectionReceiveWindow = 8 * 1024 * 1024
		quicCfg.MaxConnectionReceiveWindow = 32 * 1024 * 1024
	}

	h3 := &http3.Server{
		Addr:       cfg.Addr,
		TLSConfig:  cfg.TLSConfig,
		QUICConfig: quicCfg,
	}
	webtransport.ConfigureHTTP3Server(h3)

	server := &webtransport.Server{H3: h3}

	mux := http.NewServeMux()
	if cfg.HTTPHandler != nil {
		mux.Handle("/", cfg.HTTPHandler)
	}
	mux.HandleFunc(cfg.Path, func(w http.ResponseWriter, r *http.Request) {
		sess, err := server.Upgrade(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if cfg.Handler != nil {
			go cfg.Handler(sess.Context(), &webTransportSession{session: sess})
		}
	})
	h3.Handler = mux

	return &WebTransportServer{cfg: cfg, wt: server}, nil
}

func (s *WebTransportServer) ListenAndServe() error {
	if s.cfg.CertFile == "" || s.cfg.KeyFile == "" {
		return fmt.Errorf("cert and key are required")
	}
	return s.wt.ListenAndServeTLS(s.cfg.CertFile, s.cfg.KeyFile)
}

func (s *WebTransportServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.wt.Close()
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
