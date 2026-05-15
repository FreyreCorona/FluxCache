package network

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/FreyreCorona/FluxCache/resp"
)

// HTTP is an HTTP network transport.
type HTTP struct {
	addr string
	ln   net.Listener
	srv  *http.Server
}

// NewHTTP creates a new HTTP transport that listens on the given address.
func NewHTTP(addr string) *HTTP {
	return &HTTP{addr: addr}
}

// Listen starts the HTTP server and begins serving requests.
func (h *HTTP) Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error {
	ln, err := net.Listen("tcp", h.addr)
	if err != nil {
		return fmt.Errorf("http: listen: %w", err)
	}
	h.ln = ln
	h.srv = &http.Server{Handler: http.HandlerFunc(h.handleCommand(handlers, onWrite))}
	return h.srv.Serve(ln)
}

// Addr returns the address the server is listening on.
func (h *HTTP) Addr() net.Addr {
	if h.ln != nil {
		return h.ln.Addr()
	}
	return nil
}

// Close shuts down the HTTP server.
func (h *HTTP) Close() error {
	if h.srv != nil {
		return h.srv.Close()
	}
	return nil
}

func (h *HTTP) handleCommand(handlers map[string]HandlerFunc, onWrite WriteFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok": false, "error": "method not allowed",
			})
			return
		}

		var args []string
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok": false, "error": "invalid json",
			})
			return
		}

		if len(args) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok": false, "error": "empty command",
			})
			return
		}

		command := strings.ToUpper(args[0])
		handler, ok := handlers[command]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"ok": false, "error": fmt.Sprintf("unknown command: %s", command),
			})
			return
		}

		respArgs := make([]resp.Value, len(args)-1)
		for i, a := range args[1:] {
			respArgs[i] = resp.Value{Type: resp.TypeBulk, Bulk: a}
		}

		if onWrite != nil {
			onWrite(command, respArgs)
		}

		result := handler(respArgs)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respToJSON(result))
	}
}

func respToJSON(v resp.Value) any {
	switch v.Type {
	case resp.TypeString:
		return map[string]any{"ok": true, "value": v.Str}
	case resp.TypeBulk:
		return map[string]any{"ok": true, "value": v.Bulk}
	case resp.TypeNull:
		return map[string]any{"ok": true, "value": nil}
	case resp.TypeError:
		return map[string]any{"ok": false, "error": v.Str}
	case resp.TypeInteger:
		return map[string]any{"ok": true, "value": v.Num}
	case resp.TypeArray:
		arr := make([]any, len(v.Array))
		for i, item := range v.Array {
			arr[i] = respToJSON(item)
		}
		return map[string]any{"ok": true, "value": arr}
	default:
		return map[string]any{"ok": true, "value": nil}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
