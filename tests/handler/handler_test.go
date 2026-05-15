package handler_test

import (
	"testing"

	"github.com/FreyreCorona/FluxCache/evict"
	"github.com/FreyreCorona/FluxCache/handler"
	"github.com/FreyreCorona/FluxCache/resp"
	"github.com/FreyreCorona/FluxCache/store"
)

func BenchmarkSetGet(b *testing.B) {
	s := store.NewTTLStore(store.NewMapStore())
	s.SetEvictionPolicy(evict.NewNoEviction(), 0)
	handlers := handler.NewHandlers(s)

	b.Run("SET", func(b *testing.B) {
		args := []resp.Value{{Type: resp.TypeBulk, Bulk: "key"}, {Type: resp.TypeBulk, Bulk: "value"}}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handlers["SET"](args)
		}
	})

	b.Run("GET", func(b *testing.B) {
		args := []resp.Value{{Type: resp.TypeBulk, Bulk: "key"}}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handlers["GET"](args)
		}
	})
}
