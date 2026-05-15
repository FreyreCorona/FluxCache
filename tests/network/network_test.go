package network_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/FreyreCorona/FluxCache/network"
	"github.com/FreyreCorona/FluxCache/network/grpcpb"
	"github.com/FreyreCorona/FluxCache/resp"
	"github.com/FreyreCorona/FluxCache/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type handlerMap map[string]network.HandlerFunc

func testHandlers(s store.Store) handlerMap {
	return handlerMap{
		"PING": func(args []resp.Value) resp.Value {
			return resp.Value{Type: resp.TypeString, Str: "PONG"}
		},
		"SET": func(args []resp.Value) resp.Value {
			if len(args) < 2 {
				return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'set' command"}
			}
			s.Set(args[0].Bulk, args[1].Bulk)
			return resp.Value{Type: resp.TypeString, Str: "OK"}
		},
		"GET": func(args []resp.Value) resp.Value {
			if len(args) != 1 {
				return resp.Value{Type: resp.TypeError, Str: "ERR wrong number of arguments for 'get' command"}
			}
			val, ok := s.Get(args[0].Bulk)
			if !ok {
				return resp.Value{Type: resp.TypeNull}
			}
			return resp.Value{Type: resp.TypeBulk, Bulk: val}
		},
		"DEL": func(args []resp.Value) resp.Value {
			count := 0
			for _, arg := range args {
				_, ok := s.Get(arg.Bulk)
				if ok {
					s.Del(arg.Bulk)
					count++
				}
			}
			return resp.Value{Type: resp.TypeInteger, Num: count}
		},
	}
}

func setupNetwork(t *testing.T, n network.Network) {
	t.Helper()

	s := store.NewMapStore()
	handlers := testHandlers(s)
	onWriteCalled := make(chan struct{}, 10)
	onWrite := func(command string, args []resp.Value) {
		onWriteCalled <- struct{}{}
	}

	go func() {
		if err := n.Listen(handlers, onWrite); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	waitForAddr(t, n)
}

func waitForAddr(t *testing.T, n network.Network) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if n.Addr() != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not start listening")
}

// ─── RESP helpers ───────────────────────────────────────────────────

func respDial(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 5*time.Second)
}

func respCommand(conn net.Conn, args ...string) error {
	v := resp.Value{Type: resp.TypeArray, Array: make([]resp.Value, len(args))}
	for i, a := range args {
		v.Array[i] = resp.Value{Type: resp.TypeBulk, Bulk: a}
	}
	_, err := conn.Write(v.Marshal())
	return err
}

func respRead(conn net.Conn) (resp.Value, error) {
	rd := resp.NewResp(conn)
	return rd.Read()
}

// ─── TCP tests ──────────────────────────────────────────────────────

func TestTCPRoundtrip(t *testing.T) {
	n := network.NewTCP(":0")
	setupNetwork(t, n)
	defer n.Close()

	conn, err := respDial(n.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := respCommand(conn, "PING"); err != nil {
		t.Fatal(err)
	}
	v, err := respRead(conn)
	if err != nil || v.Type != resp.TypeString || v.Str != "PONG" {
		t.Fatalf("expected PONG, got %+v", v)
	}

	if err := respCommand(conn, "SET", "foo", "bar"); err != nil {
		t.Fatal(err)
	}
	v, err = respRead(conn)
	if err != nil || v.Type != resp.TypeString || v.Str != "OK" {
		t.Fatalf("expected OK, got %+v", v)
	}

	if err := respCommand(conn, "GET", "foo"); err != nil {
		t.Fatal(err)
	}
	v, err = respRead(conn)
	if err != nil || v.Type != resp.TypeBulk || v.Bulk != "bar" {
		t.Fatalf("expected bar, got %+v", v)
	}

	if err := respCommand(conn, "GET", "nonexistent"); err != nil {
		t.Fatal(err)
	}
	v, err = respRead(conn)
	if err != nil || v.Type != resp.TypeNull {
		t.Fatalf("expected null, got %+v", v)
	}
}

func TestTCPDel(t *testing.T) {
	n := network.NewTCP(":0")
	setupNetwork(t, n)
	defer n.Close()

	conn, err := respDial(n.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	respCommand(conn, "SET", "k", "v")
	respRead(conn)
	respCommand(conn, "DEL", "k")
	v, err := respRead(conn)
	if err != nil || v.Type != resp.TypeInteger || v.Num != 1 {
		t.Fatalf("expected 1, got %+v", v)
	}

	respCommand(conn, "DEL", "k")
	v, err = respRead(conn)
	if err != nil || v.Type != resp.TypeInteger || v.Num != 0 {
		t.Fatalf("expected 0, got %+v", v)
	}
}

// ─── HTTP tests ─────────────────────────────────────────────────────

func TestHTTPRoundtrip(t *testing.T) {
	n := network.NewHTTP(":0")
	setupNetwork(t, n)
	defer n.Close()

	addr := n.Addr().String()
	url := fmt.Sprintf("http://%s/", addr)

	do := func(args []string) map[string]interface{} {
		body, _ := json.Marshal(args)
		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return result
	}

	r := do([]string{"PING"})
	if r["ok"] != true || r["value"] != "PONG" {
		t.Fatalf("expected PONG, got %+v", r)
	}

	r = do([]string{"SET", "foo", "bar"})
	if r["ok"] != true || r["value"] != "OK" {
		t.Fatalf("expected OK, got %+v", r)
	}

	r = do([]string{"GET", "foo"})
	if r["ok"] != true || r["value"] != "bar" {
		t.Fatalf("expected bar, got %+v", r)
	}

	r = do([]string{"GET", "nonexistent"})
	if r["ok"] != true || r["value"] != nil {
		t.Fatalf("expected null, got %+v", r)
	}
}

func TestHTTPUnknownCommand(t *testing.T) {
	n := network.NewHTTP(":0")
	setupNetwork(t, n)
	defer n.Close()

	body, _ := json.Marshal([]string{"UNKNOWN"})
	resp, err := http.Post(fmt.Sprintf("http://%s/", n.Addr().String()), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&r)
	if r["ok"] != false {
		t.Fatalf("expected error, got %+v", r)
	}
}

// ─── Unix Domain Socket tests ───────────────────────────────────────

func TestUnixRoundtrip(t *testing.T) {
	path := fmt.Sprintf("%s/fluxcache_test.sock", t.TempDir())
	n := network.NewUnix(path)
	setupNetwork(t, n)
	defer n.Close()

	conn, err := net.DialTimeout("unix", path, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := respCommand(conn, "PING"); err != nil {
		t.Fatal(err)
	}
	v, err := respRead(conn)
	if err != nil || v.Type != resp.TypeString || v.Str != "PONG" {
		t.Fatalf("expected PONG, got %+v", v)
	}
}

// ─── TLS tests ──────────────────────────────────────────────────────

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

	n := network.NewTLS(":0", certFile, keyFile)
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

	if err := respCommand(conn, "PING"); err != nil {
		t.Fatal(err)
	}
	v, err := respRead(conn)
	if err != nil || v.Type != resp.TypeString || v.Str != "PONG" {
		t.Fatalf("expected PONG, got %+v", v)
	}
}

// ─── gRPC tests ─────────────────────────────────────────────────────

func TestGRPCRoundtrip(t *testing.T) {
	n := network.NewGRPC(":0")
	setupNetwork(t, n)
	defer n.Close()

	conn, err := grpc.NewClient(n.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := grpcpb.NewFluxCacheClient(conn)

	r, err := client.Exec(t.Context(), &grpcpb.Command{Name: "PING"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeString || r.Str != "PONG" {
		t.Fatalf("expected PONG, got %+v", r)
	}

	r, err = client.Exec(t.Context(), &grpcpb.Command{Name: "SET", Args: []string{"foo", "bar"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeString || r.Str != "OK" {
		t.Fatalf("expected OK, got %+v", r)
	}

	r, err = client.Exec(t.Context(), &grpcpb.Command{Name: "GET", Args: []string{"foo"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeBulk || r.Bulk != "bar" {
		t.Fatalf("expected bar, got %+v", r)
	}

	r, err = client.Exec(t.Context(), &grpcpb.Command{Name: "GET", Args: []string{"nonexistent"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeNull {
		t.Fatalf("expected null, got %+v", r)
	}
}

func TestGRPCUnknownCommand(t *testing.T) {
	n := network.NewGRPC(":0")
	setupNetwork(t, n)
	defer n.Close()

	conn, err := grpc.NewClient(n.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := grpcpb.NewFluxCacheClient(conn)
	r, err := client.Exec(t.Context(), &grpcpb.Command{Name: "UNKNOWN"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != resp.TypeError {
		t.Fatalf("expected error response, got %+v", r)
	}
}
