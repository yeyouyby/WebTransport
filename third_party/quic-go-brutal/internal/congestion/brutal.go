package congestion

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/quic-go/quic-go/internal/protocol"
)

type brutalConfig struct {
	enabled   bool
	targetBPS int64
	rtt       time.Duration
}

func loadBrutalConfig() brutalConfig {
	enabled := parseBoolEnv("QUIC_GO_BRUTAL_ENABLED", false)
	targetBPS := parseInt64Env("QUIC_GO_BRUTAL_TARGET_BPS", 50_000_000)
	rttMs := parseInt64Env("QUIC_GO_BRUTAL_RTT_MS", 120)
	if rttMs <= 0 {
		rttMs = 120
	}
	return brutalConfig{
		enabled:   enabled,
		targetBPS: targetBPS,
		rtt:       time.Duration(rttMs) * time.Millisecond,
	}
}

func brutalWindowBytes(targetBPS int64, rtt time.Duration, minWindow protocol.ByteCount) protocol.ByteCount {
	if targetBPS <= 0 {
		targetBPS = 50_000_000
	}
	if rtt <= 0 {
		rtt = 120 * time.Millisecond
	}
	b := float64(targetBPS) * rtt.Seconds() / 8.0
	if b < float64(minWindow) {
		return minWindow
	}
	if b > float64(protocol.MaxByteCount) {
		return protocol.MaxByteCount
	}
	return protocol.ByteCount(b)
}

func parseBoolEnv(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func parseInt64Env(key string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}
