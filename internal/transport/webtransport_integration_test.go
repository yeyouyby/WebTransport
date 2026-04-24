package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWebTransportReliableRoundTrip(t *testing.T) {
	t.Parallel()

	certFile, keyFile := writeSelfSignedCert(t)
	addr := pickUDPAddr(t)

	errCh := make(chan error, 1)
	server, err := NewWebTransportServer(WebTransportServerConfig{
		Addr:     addr,
		Path:     "/wt",
		CertFile: certFile,
		KeyFile:  keyFile,
		CCPolicy: CCPolicyDefault,
		Handler: func(ctx context.Context, session Session) {
			hCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			data, err := session.ReceiveReliable(hCtx)
			if err != nil {
				errCh <- err
				return
			}
			if string(data) != "ping" {
				errCh <- fmt.Errorf("unexpected payload: %s", string(data))
				return
			}
			if err := session.SendReliable(hCtx, []byte("pong")); err != nil {
				errCh <- err
				return
			}
			errCh <- nil
		},
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- server.ListenAndServe()
	}()

	t.Cleanup(func() {
		_ = server.Close()
		select {
		case err := <-serveErrCh:
			if err != nil && !isExpectedServerCloseErr(err) {
				t.Errorf("server exit error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("server close timeout")
		}
	})

	endpoint := "https://" + addr + "/wt"
	provider, err := NewWebTransportProvider(WebTransportConfig{
		EndpointURL: endpoint,
		InsecureTLS: true,
		CCPolicy:    CCPolicyDefault,
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	var sess Session
	ctxDial, cancelDial := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancelDial()
	for i := 0; i < 20; i++ {
		sess, err = provider.Open(ctxDial, "test-client")
		if err == nil {
			break
		}
		time.Sleep(80 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("open session: %v", err)
	}
	defer func() { _ = sess.Close() }()

	ctxIO, cancelIO := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelIO()
	if err := sess.SendReliable(ctxIO, []byte("ping")); err != nil {
		t.Fatalf("send ping: %v", err)
	}

	resp, err := sess.ReceiveReliable(ctxIO)
	if err != nil {
		t.Fatalf("receive pong: %v", err)
	}
	if string(resp) != "pong" {
		t.Fatalf("unexpected response: %s", string(resp))
	}

	select {
	case hErr := <-errCh:
		if hErr != nil {
			t.Fatalf("handler error: %v", hErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("handler timeout")
	}
}

func pickUDPAddr(t *testing.T) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen packet: %v", err)
	}
	addr := pc.LocalAddr().String()
	_ = pc.Close()
	return addr
}

func writeSelfSignedCert(t *testing.T) (string, string) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generate serial: %v", err)
	}
	tpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	dir := t.TempDir()
	certFile := filepath.Join(dir, "server.crt")
	keyFile := filepath.Join(dir, "server.key")

	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		_ = certOut.Close()
		t.Fatalf("encode cert: %v", err)
	}
	if err := certOut.Close(); err != nil {
		t.Fatalf("close cert file: %v", err)
	}

	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	pkcs1 := x509.MarshalPKCS1PrivateKey(priv)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: pkcs1}); err != nil {
		_ = keyOut.Close()
		t.Fatalf("encode key: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		t.Fatalf("close key file: %v", err)
	}

	return certFile, keyFile
}

func isExpectedServerCloseErr(err error) bool {
	if err == nil {
		return true
	}
	msg := err.Error()
	if strings.Contains(msg, "Server closed") {
		return true
	}
	if strings.Contains(msg, "closed network connection") {
		return true
	}
	if strings.Contains(msg, "context canceled") {
		return true
	}
	return false
}
