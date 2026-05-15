# FluxCache

Key-value store en Go inspirado en Redis, pero con un paradigma diferente.

**Filosofía:** No ser otro clon de Redis. No copiar los mismos algoritmos ni patrones con otro nombre. Aprovechar las ventajas de Go (gorutinas livianas, canales, compilación estática, despliegue con un solo binario) para construir un sistema pensado para alta concurrencia y entornos cloud de microservicios.

---

## Arquitectura de Componentes

```
┌─────────────────────────────────────────────┐
│              Network Layer                   │
│         (TCP/RESP — ya existe)               │
├─────────────────────────────────────────────┤
│              Protocol Layer                  │
│      (RESP parser/serializer — ya existe)    │
├─────────────────────────────────────────────┤
│           Command Router / Handler           │
│      (router + handlers — ya existe)         │
├────────────┬──────────────────┬──────────────┤
│   Store    │  Eviction / TTL  │ Persistence  │
│  (8 impls) │  (5 policies)    │  (5 impls)   │
├────────────┴──────────────────┴──────────────┤
│           Cluster / Replication               │
│          (visión futura)                      │
└──────────────────────────────────────────────┘
```

### Componentes por capa

**Store (8 implementaciones):** `Map`, `Sharded`, `SyncMap`, `LockFree`, `SkipList`, `BPTree`, `ART`, `CRDT`
- `OrderedStore` — `PrefixKeys` / `RangeKeys` en `SkipList`, `BPTree`, `ART`
- `TTLStore` — wrapper que agrega expiración + evicción a cualquier `Store`

**Eviction / TTL:** lazy expiration (on read) + active sweep (cada 100ms)
- Políticas: `allkeys-lru`, `allkeys-lfu`, `allkeys-random`, `volatile-ttl`, `noeviction`

**Persistence (5 implementaciones):** `AOF`, `WAL`, `RDB`, `Dual`, `Null`
- `Dual` compone dos persistences (ej: AOF + RDB)

**Protocolo:** RESP (`resp/` — parser + writer)

**Config:** YAML (`config/` + `config.yaml`) — `Load()` lee, `Build()` ensambla instancia completa

---

## Decisiones de Diseño

### Persistencia

Estado actual: **5 implementaciones** — AOF, WAL, RDB, Dual, Null.

```
                             AOF
               ┌──────────────────────────┐
               │  SET foo bar\r\n         │
               │  HSET user:1 name alice\r\n│
               │  SET baz qux\r\n         │
               └──────────────────────────┘
               ↑ append-only, fsync cada 1s
```

#### ¿Por qué no RDB puro?

Redis ofrece RDB (snapshots binarios). Es más compacto y la recuperación es más rápida que AOF puro, pero el snapshot es pesado y bloquea momentáneamente. El modo mixto de Redis (AOF con preámbulo RDB) es mejor, pero sigue siendo el mismo paradigma.

#### Alternativas evaluadas

| Enfoque | Recuperación | Tamaño disco | Impacto perf | Complejidad |
|---|---|---|---|---|
| AOF puro | Lenta (replay total) | Grande | Bajo | Baja |
| RDB | Rápida | Compacto | Alto (snapshot) | Media |
| Modo mixto (Redis) | Rápida | Compacto + incremental | Medio | Media |
| **Bitcask** | **Instantánea** | Medio | **Muy bajo** | **Baja** |
| WAL + LSM-Tree | Rápida | Muy compacto | Medio | Alta |

#### Decisión: arquitectura tipo Bitcask (futuro)

El modelo **Bitcask** (usado por Riak) es el que mejor se alinea con los objetivos de FluxCache:

1. **Un solo archivo de datos** append-only (como AOF actual)
2. **Índice en memoria** que mapea `key -> (file_id, offset, value_size, timestamp)`
3. **Escrituras:** siempre secuenciales al final del archivo → rápidas y predecibles
4. **Lecturas:** un solo seek + read → O(1) en disco
5. **Recuperación instantánea:** solo reconstruir el índice en RAM recorriendo el archivo

```
                     Bitcask
┌──────────────────────────────────┐
│         Memory (hash index)       │
│  ┌─────┐                         │
│  │ foo │──→ (offset=0, len=7)    │
│  │ bar │──→ (offset=8, len=5)    │
│  └─────┘                         │
└──────────┬───────────────────────┘
           │
┌──────────▼───────────────────────┐
│         Disk (append-only log)    │
│  ┌──────────────────────────────┐│
│  │ foo=hello │ bar=world │ ...  ││
│  └──────────────────────────────┘│
└──────────────────────────────────┘
```

Requisito: el índice debe caber en memoria. Para un cache, esto es aceptable.

**Compactación periódica** (merge): cuando el archivo crece demasiado, se escribe un nuevo archivo solo con los valores vigentes (más recientes por clave) y se reemplaza el viejo. Esto mantiene el disco acotado.

#### Por qué no LSM-Tree (RocksDB, LevelDB)

LSM-Tree es óptimo para tamaño en disco y escrituras intensivas con rangos, pero:
- Complejidad de implementación alta (Bloom filters, compaction multi-nivel, SSTables)
- Las lecturas requieren múltiples chequeos (memtable → nivel 0 → nivel 1 → ...)
- El overhead no se justifica para un cache donde la mayoría de los datos calzan en RAM

### Concurrencia

8 implementaciones de Store con distintos modelos de concurrencia:

| Store | Modelo | Ideal para |
|---|---|---|
| `MapStore` | `sync.RWMutex` global | Prototipos, baja contención |
| `ShardedStore` | `hash(key) → shard` con RWMutex propio | Alta concurrencia general |
| `SyncMapStore` | `sync.Map` | Reads-heavy, keys estables |
| `LockFreeStore` | `atomic.Pointer` + copy-on-write CAS | Escrituras esporádicas |
| `SkipListStore` | Concurrent skip list | Rangos ordenados |
| `BPTreeStore` | B+Tree con mutex por nodo | Rangos + persists |
| `ARTStore` | Adaptive Radix Tree | Prefix-heavy workloads |
| `CRDTStore` | LWW versioned pairs | Replicación futura |

### Configuración

Declarativa via YAML. El package `config/` traduce el archivo a una instancia funcionando.

```yaml
# config.yaml
server:
  port: 6379

store:
  type: map          # map | sharded | syncmap | lockfree | skiplist | bptree | art | crdt

persistence:
  type: null         # null | aof | wal | rdb | dual

eviction:
  policy: noeviction # noeviction | allkeys-lru | allkeys-lfu | allkeys-random | volatile-ttl
  maxkeys: 0
```

### Clustering / Hive (visión)

La meta es que múltiples instancias puedan conectarse entre sí formando una colmena sin un orchestrator externo. El diseño de cada nodo debe contemplar desde el inicio:

- Protocolo de gossip para descubrimiento
- Hash ring para distribución de claves (consistent hashing)
- Replicación asíncrona con conflict resolution basada en timestamp (CRDT-like)

Esto no se implementa en la fase actual, pero las decisiones de persistencia y la estructura CRDTStore lo facilitan.

---

## Estado del Proyecto

- [x] Servidor TCP con protocolo RESP (compatible con redis-cli)
- [x] Comandos: PING, SET, GET, HSET, HGET, HGETALL, EXPIRE, TTL, DEL
- [x] SET con flag `EX` (TTL en segundos)
- [x] 8 implementaciones de Store (Map, Sharded, SyncMap, LockFree, SkipList, BPTree, ART, CRDT)
- [x] OrderedStore (PrefixKeys, RangeKeys) en SkipList, BPTree, ART
- [x] TTL/Expiración lazy + active sweep
- [x] 5 Eviction policies (LRU, LFU, TTL, Random, NoEviction)
- [x] 5 Persistence implementaciones (AOF, WAL, RDB, Dual, Null)
- [x] Configuración declarativa via YAML
- [x] Makefile (fmt, vet, test, bench, build, run)
- [ ] Persistencia Bitcask
- [ ] Network Layer abstracta (TCP/HTTP/...)
- [ ] Cluster / hive mode

---

## Uso

```bash
go run .              # usa config.yaml por defecto
go run . config.yaml  # o pasar un path
```

Conectarse con redis-cli:
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
