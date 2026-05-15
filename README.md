# FluxCache

Key-value store en Go inspirado en Redis, construido con composición explícita y configuración declarativa.

---

## Arquitectura

```
┌─────────────────────────────────────────────┐
│              Network Layer                   │
│     TCP | TLS | Unix | HTTP | gRPC          │
├─────────────────────────────────────────────┤
│           Command Handlers                  │
│     PING SET GET HSET HGET HGETALL          │
│     EXPIRE TTL DEL                          │
├────────────┬──────────────────┬──────────────┤
│   Store    │  Eviction / TTL  │ Persistence  │
│  9 impls   │  5 policies      │  5 impls     │
├────────────┴──────────────────┴──────────────┤
│           Cluster / Replication               │
│          (próximo)                            │
└──────────────────────────────────────────────┘
```

## Componentes

### Store
| Store | Concurrencia | Ideal para |
|---|---|---|
| `MapStore` | `sync.RWMutex` global | Prototipos, baja contención |
| `ShardedStore` | hash → shard con RWMutex propio | Alta concurrencia general |
| `SyncMapStore` | `sync.Map` | Reads-heavy, keys estables |
| `LockFreeStore` | `atomic.Pointer` + copy-on-write CAS | Escrituras esporádicas |
| `SkipListStore` | Concurrent skip list + `OrderedStore` | Rangos ordenados |
| `BPTreeStore` | B+Tree con mutex por nodo + `OrderedStore` | Rangos + persists |
| `ARTStore` | Adaptive Radix Tree + `OrderedStore` | Prefix-heavy |
| `CRDTStore` | LWW versioned pairs | Replicación |
| `BitcaskStore` | `sync.RWMutex` + append-only log | Auto-persistente |

`TTLStore` envuelve cualquier Store sumando expiración (lazy + active sweep) y evicción.

### Eviction
- `allkeys-lru`, `allkeys-lfu`, `allkeys-random` — sobre todas las keys
- `volatile-ttl` — solo keys con TTL
- `noeviction`

### Persistence
- `AOF` — append-only log de comandos
- `WAL` — write-ahead log binario con batch commit
- `RDB` — snapshot binario periódico
- `Dual` — compone dos persistences (ej: AOF + RDB)
- `Null` — no persistencia
- `BitcaskStore` — store + persistencia en uno (no necesita persistence aparte)

### Network
| Transport | Protocolo | Uso |
|---|---|---|
| `TCP` | RESP | redis-cli, clients estándar |
| `TLS` | RESP sobre TLS | Conexiones seguras |
| `Unix` | RESP sobre Unix socket | Comunicación local |
| `HTTP` | JSON (POST /) | REST clients |
| `gRPC` | Protobuf (Exec RPC) | Microservicios |

### Config

```yaml
server:
  port: 6379
  network: tcp          # tcp | tls | unix | http | grpc
  # cert_file: server.crt
  # key_file: server.key
  # socket_path: /tmp/fluxcache.sock

store:
  type: map             # map | sharded | syncmap | lockfree | skiplist | bptree | art | crdt | bitcask
  # file: data.db       # required for bitcask
  # shard_count: 16     # for sharded / lockfree

persistence:
  type: null            # null | aof | wal | rdb | dual
  # file: database.aof
  # interval: 5s        # for rdb

eviction:
  policy: noeviction    # noeviction | allkeys-lru | allkeys-lfu | allkeys-random | volatile-ttl
  maxkeys: 0
```

## Estado

- [x] RESP protocol + comandos (PING, SET, GET, HSET, HGET, HGETALL, EXPIRE, TTL, DEL)
- [x] 9 Store implementations (Map, Sharded, SyncMap, LockFree, SkipList, BPTree, ART, CRDT, Bitcask)
- [x] OrderedStore (PrefixKeys, RangeKeys)
- [x] TTL + 5 eviction policies
- [x] 5 Persistence implementations (AOF, WAL, RDB, Dual, Null)
- [x] 5 Network transports (TCP, TLS, Unix, HTTP, gRPC)
- [x] Config YAML con validación
- [ ] Cluster / Hive (gossip, consistent hashing, replicación)

## Uso

```bash
go run .              # usa config.yaml
go run . mi-conf.yaml # o path personalizado
```

```bash
redis-cli -p 6379
> SET foo bar
OK
> GET foo
"bar"
> EXPIRE foo 60
1
> TTL foo
58
```

```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '["SET", "foo", "bar"]'
# {"ok":true,"value":"OK"}
```
