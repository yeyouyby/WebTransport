package transport

import "fmt"

type ProviderMode string

const (
	ProviderModeStub         ProviderMode = "stub"
	ProviderModeWebTransport ProviderMode = "webtransport"
)

type ProviderConfig struct {
	Mode            ProviderMode
	CCPolicy        CCPolicy
	BrutalTargetBPS int64
	BrutalRTTMs     int
	EndpointURL     string
	InsecureTLS     bool
}

func NewProvider(cfg ProviderConfig) (Provider, error) {
	switch cfg.Mode {
	case "", ProviderModeStub:
		return NewStubProvider(cfg.CCPolicy, cfg.BrutalTargetBPS, cfg.BrutalRTTMs), nil
	case ProviderModeWebTransport:
		return NewWebTransportProvider(WebTransportConfig{
			EndpointURL:     cfg.EndpointURL,
			InsecureTLS:     cfg.InsecureTLS,
			CCPolicy:        cfg.CCPolicy,
			BrutalTargetBPS: cfg.BrutalTargetBPS,
			BrutalRTTMs:     cfg.BrutalRTTMs,
		})
	default:
		return nil, fmt.Errorf("unknown provider mode: %s", cfg.Mode)
	}
}
