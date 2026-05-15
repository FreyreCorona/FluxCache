package network_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/FreyreCorona/FluxCache/network"
)

func TestUnixRoundtrip(t *testing.T) {
	path := fmt.Sprintf("%s/fluxcache_test.sock", t.TempDir())
	n := network.NewUnix(path, 0)
	setupNetwork(t, n)
	defer n.Close()

	conn, err := net.DialTimeout("unix", path, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	v := mustResp(conn, "PING")
	assertPONG(t, v)
}
