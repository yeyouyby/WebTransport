package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"matrix-gateway/internal/storage"
)

type Config struct {
	Transport TransportConfig
	Storage   StorageConfig
	Fallback  FallbackConfig
	Crypto    CryptoConfig
	Protocol  ProtocolConfig
	Prefetch  PrefetchConfig
	Sharding  ShardingConfig
}

type TransportConfig struct {
	CCPolicy        string
	BrutalTargetBPS int64
	BrutalRTTMs     int
	Provider        string
	EndpointURL     string
	InsecureTLS     bool
	ServerEnabled   bool
	ServerAddr      string
	ServerPath      string
	ServerCertFile  string
	ServerKeyFile   string
}

type StorageConfig struct {
	GDriveBaseURL string
	GDriveFileID  string
	MaxRetries    int
	BaseBackoffMs int
	SAEntries     []storage.SAEntry
}

type FallbackConfig struct {
	Addr            string
	CertMap         map[string]string
	ShutdownTimeout time.Duration
}

type CryptoConfig struct {
	KeyHex   string
	NonceHex string
}

type ProtocolConfig struct {
	TokenSecret string
	HeaderTTL   time.Duration
}

type PrefetchConfig struct {
	Enabled    bool
	MaxEntries int
	TTLSeconds int
}

type ShardingConfig struct {
	ImageHosts    []string
	VideoHosts    []string
	AudioHosts    []string
	DefaultScheme string
}

func LoadFromEnv() Config {
	return Config{
		Transport: TransportConfig{
			CCPolicy:        envOrDefault("CC_POLICY", "default"),
			BrutalTargetBPS: int64(envInt("BRUTAL_TARGET_BPS", 50_000_000)),
			BrutalRTTMs:     envInt("BRUTAL_RTT_MS", 120),
			Provider:        envOrDefault("TRANSPORT_PROVIDER", "stub"),
			EndpointURL:     envOrDefault("TRANSPORT_ENDPOINT_URL", "https://127.0.0.1:8444/wt"),
			InsecureTLS:     envBool("TRANSPORT_INSECURE_TLS", true),
			ServerEnabled:   envBool("TRANSPORT_SERVER_ENABLED", false),
			ServerAddr:      envOrDefault("TRANSPORT_SERVER_ADDR", ":8444"),
			ServerPath:      envOrDefault("TRANSPORT_SERVER_PATH", "/wt"),
			ServerCertFile:  envOrDefault("TRANSPORT_SERVER_CERT_FILE", ""),
			ServerKeyFile:   envOrDefault("TRANSPORT_SERVER_KEY_FILE", ""),
		},
		Storage: StorageConfig{
			GDriveBaseURL: envOrDefault("GDRIVE_BASE_URL", "https://www.googleapis.com/drive/v3/files"),
			GDriveFileID:  envOrDefault("GDRIVE_FILE_ID", ""),
			MaxRetries:    envInt("GDRIVE_MAX_RETRIES", 4),
			BaseBackoffMs: envInt("GDRIVE_BASE_BACKOFF_MS", 80),
			SAEntries:     parseSAEntries(envOrDefault("SA_ENTRIES", "sa-a:1:20,sa-b:1:20")),
		},
		Fallback: FallbackConfig{
			Addr:            envOrDefault("FALLBACK_ADDR", ":8443"),
			CertMap:         parseCertMap(envOrDefault("TLS_CERT_MAP", "")),
			ShutdownTimeout: time.Duration(envInt("SHUTDOWN_TIMEOUT_SEC", 5)) * time.Second,
		},
		Crypto: CryptoConfig{
			KeyHex:   envOrDefault("CRYPTO_KEY_HEX", ""),
			NonceHex: envOrDefault("CRYPTO_NONCE_HEX", ""),
		},
		Protocol: ProtocolConfig{
			TokenSecret: envOrDefault("PROTOCOL_TOKEN_SECRET", "dev-secret"),
			HeaderTTL:   time.Duration(envInt("PROTOCOL_HEADER_TTL_SEC", 30)) * time.Second,
		},
		Prefetch: PrefetchConfig{
			Enabled:    envBool("PREFETCH_ENABLED", true),
			MaxEntries: envInt("PREFETCH_MAX_ENTRIES", 128),
			TTLSeconds: envInt("PREFETCH_TTL_SEC", 20),
		},
		Sharding: ShardingConfig{
			ImageHosts:    parseCSV(envOrDefault("IMAGE_SHARD_HOSTS", "s1.api.com,s2.api.com")),
			VideoHosts:    parseCSV(envOrDefault("VIDEO_SHARD_HOSTS", "s1.api.com,s2.api.com,s3.api.com")),
			AudioHosts:    parseCSV(envOrDefault("AUDIO_SHARD_HOSTS", "s1.api.com,s2.api.com,s3.api.com")),
			DefaultScheme: envOrDefault("SHARD_SCHEME", "https"),
		},
	}
}

func parseCSV(raw string) []string {
	items := strings.Split(raw, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func parseSAEntries(raw string) []storage.SAEntry {
	parts := strings.Split(raw, ",")
	out := make([]storage.SAEntry, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		cols := strings.Split(part, ":")
		if len(cols) != 3 {
			continue
		}
		weight, errW := strconv.Atoi(cols[1])
		qps, errQ := strconv.Atoi(cols[2])
		if errW != nil || errQ != nil {
			continue
		}
		out = append(out, storage.SAEntry{
			ID:     cols[0],
			Weight: weight,
			MaxQPS: qps,
			Token:  "",
		})
	}
	if len(out) == 0 {
		out = []storage.SAEntry{{ID: "default", Weight: 1, MaxQPS: 20}}
	}
	return out
}

func parseCertMap(raw string) map[string]string {
	out := make(map[string]string)
	if raw == "" {
		return out
	}
	items := strings.Split(raw, ",")
	for _, item := range items {
		cols := strings.Split(item, "=")
		if len(cols) != 2 {
			continue
		}
		host := strings.TrimSpace(cols[0])
		bundle := strings.TrimSpace(cols[1])
		if host == "" || bundle == "" {
			continue
		}
		out[host] = bundle
	}
	return out
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	if v == "1" || v == "true" || v == "yes" || v == "y" || v == "on" {
		return true
	}
	if v == "0" || v == "false" || v == "no" || v == "n" || v == "off" {
		return false
	}
	return fallback
}
