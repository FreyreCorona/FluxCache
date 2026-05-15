package network

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/FreyreCorona/FluxCache/resp"
)

type HTTP struct {
	addr string
	srv  *http.Server
}

func NewHTTP(addr string) *HTTP {
	return &HTTP{addr: addr}
}

func (h *HTTP) Listen(handlers map[string]HandlerFunc, onWrite WriteFunc) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleCommand(handlers, onWrite))
	h.srv = &http.Server{Addr: h.addr, Handler: mux}
	return h.srv.ListenAndServe()
}

func (h *HTTP) Close() error {
	if h.srv != nil {
		return h.srv.Close()
	}
	return nil
}

func (h *HTTP) handleCommand(handlers map[string]HandlerFunc, onWrite WriteFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"ok": false, "error": "method not allowed",
			})
			return
		}

		var args []string
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"ok": false, "error": "invalid json",
			})
			return
		}

		if len(args) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"ok": false, "error": "empty command",
			})
			return
		}

		command := strings.ToUpper(args[0])
		handler, ok := handlers[command]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]interface{}{
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

func respToJSON(v resp.Value) interface{} {
	switch v.Type {
	case resp.TypeString:
		return map[string]interface{}{"ok": true, "value": v.Str}
	case resp.TypeBulk:
		return map[string]interface{}{"ok": true, "value": v.Bulk}
	case resp.TypeNull:
		return map[string]interface{}{"ok": true, "value": nil}
	case resp.TypeError:
		return map[string]interface{}{"ok": false, "error": v.Str}
	case resp.TypeInteger:
		return map[string]interface{}{"ok": true, "value": v.Num}
	case resp.TypeArray:
		arr := make([]interface{}, len(v.Array))
		for i, item := range v.Array {
			arr[i] = respToJSON(item)
		}
		return map[string]interface{}{"ok": true, "value": arr}
	default:
		return map[string]interface{}{"ok": true, "value": nil}
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
