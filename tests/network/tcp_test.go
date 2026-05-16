package network_test

import (
	"testing"

	"github.com/FreyreCorona/FluxCache/network"
)

func TestTCPRoundtrip(t *testing.T) {
	n := network.NewTCP(":0", 0, "")
	setupNetwork(t, n)
	defer n.Close()

	conn, err := respDial(n.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	v := mustResp(conn, "PING")
	assertPONG(t, v)

	v = mustResp(conn, "SET", "foo", "bar")
	assertOK(t, v)

	v = mustResp(conn, "GET", "foo")
	assertBulk(t, v, "bar")

	v = mustResp(conn, "GET", "nonexistent")
	assertNull(t, v)
}

func TestTCPDel(t *testing.T) {
	n := network.NewTCP(":0", 0, "")
	setupNetwork(t, n)
	defer n.Close()

	conn, err := respDial(n.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	mustResp(conn, "SET", "k", "v")
	v := mustResp(conn, "DEL", "k")
	assertInt(t, v, 1)

	v = mustResp(conn, "DEL", "k")
	assertInt(t, v, 0)
}

func BenchmarkTCPRoundtrip(b *testing.B) {
	n := setupNetworkB(b, network.NewTCP(":0", 0, ""))
	defer n.Close()

	c, err := respDial(n.Addr().String())
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()

	mustResp(c, "SET", "benchkey", "benchval")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mustResp(c, "GET", "benchkey")
	}
}
