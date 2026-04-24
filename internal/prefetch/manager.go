package prefetch

import (
	"sync"
	"time"
)

type cacheKey struct {
	offset uint64
	length uint32
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type Manager struct {
	mu         sync.RWMutex
	maxEntries int
	ttl        time.Duration
	items      map[cacheKey]cacheEntry
}

func NewManager(maxEntries int, ttl time.Duration) *Manager {
	if maxEntries <= 0 {
		maxEntries = 64
	}
	if ttl <= 0 {
		ttl = 20 * time.Second
	}
	return &Manager{
		maxEntries: maxEntries,
		ttl:        ttl,
		items:      make(map[cacheKey]cacheEntry),
	}
}

func (m *Manager) Get(offset uint64, length uint32) ([]byte, bool) {
	key := cacheKey{offset: offset, length: length}
	now := time.Now()

	m.mu.RLock()
	entry, ok := m.items[key]
	m.mu.RUnlock()
	if !ok || now.After(entry.expiresAt) {
		if ok {
			m.mu.Lock()
			delete(m.items, key)
			m.mu.Unlock()
		}
		return nil, false
	}
	out := make([]byte, len(entry.data))
	copy(out, entry.data)
	return out, true
}

func (m *Manager) Put(offset uint64, length uint32, data []byte) {
	if len(data) == 0 {
		return
	}
	key := cacheKey{offset: offset, length: length}
	copyData := make([]byte, len(data))
	copy(copyData, data)

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.items) >= m.maxEntries {
		m.evictOneLocked()
	}
	m.items[key] = cacheEntry{data: copyData, expiresAt: time.Now().Add(m.ttl)}
}

func (m *Manager) evictOneLocked() {
	for k := range m.items {
		delete(m.items, k)
		return
	}
}
