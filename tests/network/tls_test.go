package network_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"github.com/FreyreCorona/FluxCache/network"
)

func generateTestCert(t *testing.T) (certPem, keyPem []byte) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	certPem = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	keyPem = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	return
}

func writeTempCert(t *testing.T, certPem, keyPem []byte) (certFile, keyFile string) {
	t.Helper()

	dir := t.TempDir()
	certFile = dir + "/server.crt"
	keyFile = dir + "/server.key"
	if err := os.WriteFile(certFile, certPem, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, keyPem, 0644); err != nil {
		t.Fatal(err)
	}
	return
}

func TestTLSRoundtrip(t *testing.T) {
	certPem, keyPem := generateTestCert(t)
	certFile, keyFile := writeTempCert(t, certPem, keyPem)

	n := network.NewTLS(":0", certFile, keyFile, 0)
	setupNetwork(t, n)
	defer n.Close()

	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", n.Addr().String(), &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	v := mustResp(conn, "PING")
	assertPONG(t, v)
}
