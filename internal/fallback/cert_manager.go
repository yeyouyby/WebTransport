package fallback

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var ErrNoExactCert = errors.New("no exact cert for server name")

type CertManager struct {
	mu    sync.RWMutex
	certs map[string]*tls.Certificate
}

func NewCertManager() *CertManager {
	return &CertManager{certs: make(map[string]*tls.Certificate)}
}

func (m *CertManager) LoadExact(host, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("load cert %s: %w", host, err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.certs[strings.ToLower(strings.TrimSpace(host))] = &cert
	return nil
}

func (m *CertManager) SetExact(host string, cert tls.Certificate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.certs[strings.ToLower(strings.TrimSpace(host))] = &cert
}

func (m *CertManager) GetExactCert(serverName string) (*tls.Certificate, error) {
	name := strings.ToLower(strings.TrimSpace(serverName))
	m.mu.RLock()
	defer m.mu.RUnlock()
	cert, ok := m.certs[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoExactCert, serverName)
	}
	return cert, nil
}
