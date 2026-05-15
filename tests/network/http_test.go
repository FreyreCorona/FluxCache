package network_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/FreyreCorona/FluxCache/network"
)

func TestHTTPRoundtrip(t *testing.T) {
	n := network.NewHTTP(":0")
	setupNetwork(t, n)
	defer n.Close()

	url := fmt.Sprintf("http://%s/", n.Addr().String())

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
