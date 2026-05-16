# FluxCache

Key-value store in Go inspired by Redis, built with explicit composition and declarative configuration.

---

## Architecture

```
┌─────────────────────────────────────────────┐
│              Network Layer                  │
│     TCP | TLS | Unix | HTTP | gRPC          │
├─────────────────────────────────────────────┤
│           Command Handlers                  │
│     PING SET GET HSET HGET HGETALL          │
│     EXPIRE TTL DEL                          │
├────────────┬──────────────────┬─────────────┤
│   Store    │  Eviction / TTL  │ Persistence │
│  9 impls   │  5 policies      │  5 impls    │
├────────────┴──────────────────┴─────────────┤
│           Cluster / Replication             │
│          (next)                             │
└─────────────────────────────────────────────┘
```

## Components

### Store
| Store | Concurrency | Ideal for |
|---|---|---|
| `MapStore` | Global `sync.RWMutex` | Prototypes, low contention |
| `ShardedStore` | hash → shard with own RWMutex | General high concurrency |
| `SyncMapStore` | `sync.Map` | Reads-heavy, stable keys |
| `LockFreeStore` | `atomic.Pointer` + copy-on-write CAS | Sparse writes |
| `SkipListStore` | Concurrent skip list + `OrderedStore` | Sorted ranges |
| `BPTreeStore` | B+Tree with per-node mutex + `OrderedStore` | Ranges + persistence |
| `ARTStore` | Adaptive Radix Tree + `OrderedStore` | Prefix-heavy |
| `CRDTStore` | LWW versioned pairs | Replication |
| `BitcaskStore` | `sync.RWMutex` + append-only log | Auto-persistent |

`TTLStore` wraps any Store adding expiration (lazy + active sweep) and eviction.

### Eviction
- `allkeys-lru`, `allkeys-lfu`, `allkeys-random` — over all keys
- `volatile-ttl` — only keys with TTL
- `noeviction`

### Persistence
- `AOF` — append-only command log
- `WAL` — binary write-ahead log with batch commit
- `RDB` — periodic binary snapshot
- `Dual` — composes two persistences (e.g. AOF + RDB)
- `Null` — no persistence
- `BitcaskStore` — store + persistence in one (no separate persistence needed)

### Network
| Transport | Protocol | Usage |
|---|---|---|
| `TCP` | RESP | redis-cli, standard clients |
| `TLS` | RESP over TLS | Secure connections |
| `Unix` | RESP over Unix socket | Local communication |
| `HTTP` | JSON (POST /) | REST clients |
| `gRPC` | Protobuf (Exec RPC) | Microservices |

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

## Status

- [x] RESP protocol + commands (PING, SET, GET, HSET, HGET, HGETALL, EXPIRE, TTL, DEL)
- [x] 9 Store implementations (Map, Sharded, SyncMap, LockFree, SkipList, BPTree, ART, CRDT, Bitcask)
- [x] OrderedStore (PrefixKeys, RangeKeys)
- [x] TTL + 5 eviction policies
- [x] 5 Persistence implementations (AOF, WAL, RDB, Dual, Null)
- [x] 5 Network transports (TCP, TLS, Unix, HTTP, gRPC)
- [x] YAML config with validation
- [ ] Cluster / Hive (gossip, consistent hashing, replication)

## Usage

```bash
go run .              # uses config.yaml
go run . my-conf.yaml # or custom path
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
