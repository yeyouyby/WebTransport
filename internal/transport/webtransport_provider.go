package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"

	quic "github.com/quic-go/quic-go"
	webtransport "github.com/quic-go/webtransport-go"
)

type WebTransportConfig struct {
	EndpointURL     string
	InsecureTLS     bool
	CCPolicy        CCPolicy
	BrutalTargetBPS int64
	BrutalRTTMs     int
}

type WebTransportProvider struct {
	cfg    WebTransportConfig
	dialer *webtransport.Dialer
}

func NewWebTransportProvider(cfg WebTransportConfig) (*WebTransportProvider, error) {
	if cfg.EndpointURL == "" {
		return nil, fmt.Errorf("empty webtransport endpoint")
	}
	if _, err := url.Parse(cfg.EndpointURL); err != nil {
		return nil, fmt.Errorf("invalid endpoint url: %w", err)
	}
	u, _ := url.Parse(cfg.EndpointURL)
	if u.Scheme != "https" {
		return nil, fmt.Errorf("endpoint must use https scheme")
	}
	if cfg.BrutalTargetBPS <= 0 {
		cfg.BrutalTargetBPS = 50_000_000
	}
	if cfg.BrutalRTTMs <= 0 {
		cfg.BrutalRTTMs = 120
	}

	qcfg := &quic.Config{EnableDatagrams: true, EnableStreamResetPartialDelivery: true}
	if cfg.CCPolicy == CCPolicyBrutal {
		qcfg.InitialConnectionReceiveWindow = 8 * 1024 * 1024
		qcfg.MaxConnectionReceiveWindow = 32 * 1024 * 1024
	}

	dialer := &webtransport.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureTLS,
			NextProtos:         []string{"h3"},
		},
		QUICConfig: qcfg,
	}

	return &WebTransportProvider{cfg: cfg, dialer: dialer}, nil
}

func (p *WebTransportProvider) Open(ctx context.Context, clientID string) (Session, error) {
	endpoint := endpointWithClientID(p.cfg.EndpointURL, clientID)
	resp, sess, err := p.dialer.Dial(ctx, endpoint, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("dial webtransport: %w", err)
	}
	if resp != nil && (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices) {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	if sess == nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("webtransport session is nil")
	}
	var body io.Closer
	if resp != nil {
		body = resp.Body
	}
	return &webTransportSession{session: sess, respBody: body}, nil
}

func endpointWithClientID(base, clientID string) string {
	u, err := url.Parse(base)
	if err != nil || clientID == "" {
		return base
	}
	q := u.Query()
	q.Set("client_id", clientID)
	u.RawQuery = q.Encode()
	return u.String()
}

type webTransportSession struct {
	session  *webtransport.Session
	respBody io.Closer
}

const maxReliableFrameBytes = 1 * 1024 * 1024

func (s *webTransportSession) SendReliable(ctx context.Context, payload []byte) error {
	if len(payload) > maxReliableFrameBytes {
		return fmt.Errorf("reliable frame too large: %d > %d", len(payload), maxReliableFrameBytes)
	}

	stream, err := s.session.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	if _, err := stream.Write(payload); err != nil {
		_ = stream.Close()
		return fmt.Errorf("write stream: %w", err)
	}
	if err := stream.Close(); err != nil {
		return fmt.Errorf("close stream: %w", err)
	}
	return nil
}

func (s *webTransportSession) ReceiveReliable(ctx context.Context) ([]byte, error) {
	stream, err := s.session.AcceptStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("accept stream: %w", err)
	}
	defer stream.Close()
	b, err := io.ReadAll(io.LimitReader(stream, maxReliableFrameBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}
	if len(b) > maxReliableFrameBytes {
		return nil, fmt.Errorf("reliable frame too large: %d > %d", len(b), maxReliableFrameBytes)
	}
	return b, nil
}

func (s *webTransportSession) SendUnreliable(_ context.Context, payload []byte) error {
	if err := s.session.SendDatagram(payload); err != nil {
		return fmt.Errorf("send datagram: %w", err)
	}
	return nil
}

func (s *webTransportSession) ReceiveUnreliable(ctx context.Context) ([]byte, error) {
	b, err := s.session.ReceiveDatagram(ctx)
	if err != nil {
		return nil, fmt.Errorf("receive datagram: %w", err)
	}
	return b, nil
}

func (s *webTransportSession) Close() error {
	if err := s.session.CloseWithError(0, ""); err != nil {
		return err
	}
	if s.respBody != nil {
		if err := s.respBody.Close(); err != nil {
			return err
		}
	}
	return nil
}
