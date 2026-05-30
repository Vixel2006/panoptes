# architecture

panoptes follows a strict **hexagonal architecture** (ports and adapters). the core domain knows nothing about the TUI, SQLite, TLS, or the network. everything is wired together at the top level in `cmd/main.go`.

```
                     ┌──────────────────────────────────────┐
                     │          cmd/main.go                 │
                     │       (composition root)             │
                     └──────┬──────┬──────┬──────┬──────────┘
                            │      │      │      │
                ┌───────────┘      │      │      └────────────┐
                ▼                  ▼      ▼                   ▼
        ┌───────────────┐  ┌──────────────┐  ┌────────────────────┐
        │   Adapters    │  │  App Layer   │  │  Infrastructure    │
        │  (inbound)    │──┤  (use cases) │  │  (outbound infra)  │
        │               │  │              │  │                    │
        │ Intercept     │  │ Interceptor  │  │  infra/tls/        │
        │ Adapter       │  │ SessionMgr   │  │  infra/db/         │
        │ (MITM proxy)  │  │ GroupMgr     │  │  infra/transport/  │
        │               │  │ NoteMgr      │  │                    │
        └───────┬───────┘  └──────┬───────┘  └────────────────────┘
                │                 │                    │
                │     ┌───────────┘                    │
                │     ▼                                │
                │  ┌──────────────────────────────────────┐
                │  │            Core Domain               │
                │  │  ┌──────────┐  ┌──────────────────┐  │
                │  │  │  models  │  │      ports       │  │
                │  │  │(entities)│  │  (interfaces)    │  │
                │  │  └──────────┘  └──────────────────┘  │
                │  │  ┌──────────────────────────────────┐│
                │  │  │     services (pure domain)       ││
                │  │  │     Barrier                      ││
                │  │  └──────────────────────────────────┘│
                │  └──────────────────────────────────────┘
                │
                ▼
        ┌───────────────┐
        │   Adapters    │
        │  (outbound)   │
        │               │
        │  repo/        │  ← SQLite repositories
        │  decompressor │  ← compression codecs
        │  uuid         │  ← ID generation
        └───────────────┘
```

## layers

### core domain (`internal/core/`)

the heart of the application. zero imports from outside `core/`.

- **`core/models/`** — pure data structs: `Session`, `Group`, `Request`, `Response`, `Note`. no HTTP types, no serialisation logic, no infrastructure concerns.
- **`core/ports/`** — interface contracts that the core defines and adapters implement:
  - `SessionRepo`, `GroupRepo`, `NoteRepo`, `RequestRepo`, `ResponseRepo` — persistence ports
  - `SessionUseCase`, `GroupUseCase`, `NoteUseCase` — application boundary ports
  - `InterceptorPort`, `BarrierPort` — interceptor control ports
  - `IDGenerator` — identity generation
  - `Decompressor` — content decoding
  - `RequestWriter`, `ResponseWriter` — write-only persistence (for async logging)
- **`core/services/`** — pure domain services (currently just `Barrier`, a synchronisation primitive for traffic breakpointing)

### application layer (`internal/app/`)

thin orchestration layer that wires ports into use cases. depends on `core/models` and `core/ports` only.

- **`app/interceptor.go`** — receives captured domain models, pushes them to the TUI channel, persists them asynchronously via a background worker
- **`app/session.go`** — `SessionManager`: create, load, list, rename, delete sessions
- **`app/group.go`** — `GroupManager`: create, get, list, delete groups
- **`app/note.go`** — `NoteManager`: create, list, update, delete notes

### adapters (`internal/adapters/`)

bridge between the core and the outside world.

**inbound adapters** (driving side):
- **`adapters/interceptor.go`** — the MITM proxy engine. handles raw TCP connections, TLS termination, HTTP parsing, delegates to the interceptor and barrier ports
- **`ui/`** — the Bubble Tea TUI (user-facing presentation layer)

**outbound adapters** (driven side):
- **`adapters/repo/`** — SQLite implementations of `SessionRepo`, `GroupRepo`, `NoteRepo`, `RequestRepo`, `ResponseRepo`
- **`adapters/decompressor.go`** — gzip/deflate/brotli body decompression
- **`adapters/uuid.go`** — UUID-based ID generation

### infrastructure (`internal/infra/`)

low-level technical plumbing.

- **`infra/db/`** — SQLite connection setup, migrations
- **`infra/tls/`** — root CA generation (RSA 2048), leaf certificate signing per hostname
- **`infra/transport/`** — raw TCP server that accepts connections and delegates to the adapter's `HandleConn`

## dependency rules

```
core/models → nothing (zero dependencies)
core/ports  → core/models only
core/services → core/models + core/ports
app/        → core/models + core/ports
adapters/   → core/models + core/ports + core/services + infra/
infra/      → nothing (standalone)
ui/         → core/models + core/ports
cmd/main.go → everything (wiring)
```

if a file imports from a layer it shouldn't, that's a bug.

## data flow

1. browser sends request → `transport.Server` accepts TCP connection
2. `InterceptAdapter.HandleConn` reads the first byte, routes to the correct handler (HTTP/CONNECT/TLS)
3. handler reads the HTTP request, decompresses the body, builds a `model.Request`
4. `InterceptorPort.InterceptRequest(model.Request)` sends a snapshot to the TUI and queues it for async persistence
5. `BarrierPort.Lock()` blocks the goroutine if interceptor mode is on
6. once released, `transport.RoundTrip()` forwards the request upstream
7. upstream response is read, decompressed, built into `model.Response`
8. `InterceptorPort.InterceptResponse(model.Response)` queues for persistence
9. response is written back to the client
10. background worker drains the persistence channels and writes to SQLite

## key design decisions

- **write-only persistence ports** (`RequestWriter`, `ResponseWriter`) are separate from read ports (`SessionRepo`, etc.). the interceptor never reads — it only writes. reads go through the UI layer directly to the repo.
- **async persistence** uses buffered channels (1024 capacity) with a `default:` fallback. if the worker is backed up, new events are silently dropped rather than blocking the proxy.
- **no framework** — manual dependency injection in `cmd/main.go`. no magic, no init() functions, no service locators.
