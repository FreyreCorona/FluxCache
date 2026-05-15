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
│  (mapas +  │  (políticas de   │  (AOF →      │
│   shards)  │   expulsión)     │   Bitcask)   │
├────────────┴──────────────────┴──────────────┤
│           Cluster / Replication               │
│          (visión futura)                      │
└──────────────────────────────────────────────┘
```

### Componentes para desarrollar (por orden de dependencia)

1. **Store Engine** — estructura de datos concurrente que guarda key-value. Hoy son `map[string]string` sueltos con mutex global. Evolucionar a **sharded design** (hash del key → bucket con su propio RWMutex).

2. **TTL / Expiración** — timer por clave con limpieza lazy (lectura de clave expirada) + active expiration (barrido periódico).

3. **Eviction Policy** — política de desalojo al alcanzar `max_memory`: allkeys-lru, volatile-ttl, none.

4. **Persistence Engine** — abstracción `Persistence` con implementations concretas: AOF (actual), Bitcask (próximo).

5. **Config System** — loader de YAML/TOML que traduzca configuración a parámetros del engine.

6. **Cluster / Hive** — gossip, consistent hashing, replicación. Las interfaces conviene pensarlas desde ahora para evitar refactors grandes después.

---

## Decisiones de Diseño

### Persistencia

Estado actual: **AOF (Append-Only File)** — log de comandos en formato RESP.

```
                             AOF actual
               ┌──────────────────────────┐
               │  SET foo bar\r\n         │
               │  HSET user:1 name alice\r\n│
               │  SET baz qux\r\n         │
               └──────────────────────────┘
               ↑ append-only, fsync cada 1s
```

#### ¿Por qué no RDB?

Redis ofrece RDB (snapshots binarios). Es más compacto y la recuperación es más rápida que AOF puro, pero el snapshot es pesado y bloquea momentáneamente. El modo mixto de Redis (AOF con preámbulo RDB) es mejor, pero sigue siendo el mismo paradigma.

#### Alternativas evaluadas

| Enfoque | Recuperación | Tamaño disco | Impacto perf | Complejidad |
|---|---|---|---|---|
| AOF puro (actual) | Lenta (replay total) | Grande | Bajo | Baja |
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

Estado actual: `sync.RWMutex` por estructura de datos.

Estrategia definida: **sharded mutexes** (hash del key → bucket con su propio RWMutex) para reducir contención en workloads intensivos.

```
┌────────────┬────────────┬────────────┐
│  shard 0   │  shard 1   │  shard 2   │
│ ┌─────┐   │ ┌─────┐   │ ┌─────┐   │
│ │ map │   │ │ map │   │ │ map │   │
│ │ mutex│   │ │ mutex│   │ │ mutex│   │
│ └─────┘   │ └─────┘   │ └─────┘   │
└────────────┴────────────┴────────────┘
```

A futuro: explorar **estructuras lock-free** (mapas concurrentes tipo `sync.Map` o `xsync.MapOf`) para paths críticos donde la contención sea un problema medible.

### Configuración

Declarativa y cloud-native (YAML/TOML), no comandos en runtime como Redis CONFIG SET.

```yaml
# fluxcache.yaml
server:
  host: "0.0.0.0"
  port: 6379

persistence:
  engine: "bitcask" # aof | bitcask
  sync_interval: 1s
  compaction:
    threshold_mb: 1024
    cron: "0 3 * * *"

storage:
  max_memory: "2gb"
  eviction: "allkeys-lru" # none | allkeys-lru | volatile-ttl

replication:
  mode: "standalone" # standalone | leader | follower
```

### Clustering / Hive (visión)

La meta es que múltiples instancias puedan conectarse entre sí formando una colmena sin un orchestrator externo. El diseño de cada nodo debe contemplar desde el inicio:

- Protocolo de gossip para descubrimiento
- Hash ring para distribución de claves (consistent hashing)
- Replicación asíncrona con conflict resolution basada en timestamp (CRDT-like)

Esto no se implementa en la fase actual, pero las decisiones de persistencia (Bitcask) y la estructura de datos lo facilitan: cada clave tiene un timestamp, lo que hace trivial resolver conflictos por "último escritor gana".

---

## Estado del Proyecto

Prototipo temprano. Funcionalidades actuales:
- [x] Servidor TCP con protocolo RESP (compatible con redis-cli)
- [x] Comandos: PING, SET, GET, HSET, HGET, HGETALL
- [x] Persistencia AOF básica
- [ ] Persistencia Bitcask
- [ ] Sharded mutexes
- [ ] Configuración declarativa
- [ ] Expiración de claves (TTL)
- [ ] Eviction policies
- [ ] Cluster / hive mode

---

## Uso

```bash
go run .
```

Conectarse con redis-cli:
```bash
redis-cli -p 6379
> SET foo bar
OK
> GET foo
"bar"
```
