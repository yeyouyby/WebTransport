package sharding

import (
	"fmt"
	"hash/fnv"
)

const (
	ResourceImage = "image"
	ResourceVideo = "video"
	ResourceAudio = "audio"
)

type Config struct {
	ImageHosts    []string
	VideoHosts    []string
	AudioHosts    []string
	DefaultScheme string
}

type Manager struct {
	cfg Config
}

func NewManager(cfg Config) (*Manager, error) {
	if len(cfg.ImageHosts) == 0 {
		return nil, fmt.Errorf("image shard hosts is empty")
	}
	if len(cfg.VideoHosts) == 0 {
		return nil, fmt.Errorf("video shard hosts is empty")
	}
	if len(cfg.AudioHosts) == 0 {
		return nil, fmt.Errorf("audio shard hosts is empty")
	}
	if cfg.DefaultScheme == "" {
		cfg.DefaultScheme = "https"
	}
	return &Manager{cfg: cfg}, nil
}

func (m *Manager) PickHost(resourceType string, key string) (string, error) {
	hosts, err := m.hostsByType(resourceType)
	if err != nil {
		return "", err
	}
	if key == "" {
		return hosts[0], nil
	}

	bestHost := hosts[0]
	bestScore := uint64(0)
	for _, host := range hosts {
		score := hashScore(host + ":" + key)
		if score > bestScore {
			bestScore = score
			bestHost = host
		}
	}
	return bestHost, nil
}

func (m *Manager) Config() Config {
	return m.cfg
}

func (m *Manager) hostsByType(resourceType string) ([]string, error) {
	switch resourceType {
	case ResourceImage:
		return m.cfg.ImageHosts, nil
	case ResourceVideo:
		return m.cfg.VideoHosts, nil
	case ResourceAudio:
		return m.cfg.AudioHosts, nil
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

func hashScore(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
