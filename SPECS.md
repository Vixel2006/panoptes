# Project Specification: Argus Penoptes (Penoptes)

**Sub-title:** A high-performance, open-source intercepting proxy and web fuzzer for the security community.
**Language:** Go (Golang)
**Interface:** Hybrid (Terminal User Interface via Bubble Tea + Local REST/WebSocket API)

## 1. Core Architecture Design

`optic` must run with a strictly decoupled architecture. The core networking engine handles the wire speed, while a decoupled storage and UI engine processes presentation states.

```
                  [ Web Browser / Client ]
                             │ (HTTP/1.1, HTTP/2 via TLS)
                             ▼
              ┌─────────────────────────────┐
              │  Optic Core Proxy Engine    │
              │  (Go Channels & Mutexes)    │
              └──────────────┬──────────────┘
                             │
              ┌──────────────┴──────────────┐
              │      SQLite Storage         │
              │  (WAL Mode, Async Writer)   │
              └──────────────┬──────────────┘
                             │
              ┌──────────────┴──────────────┐
              │    Presentation Interface   │
              │  (Bubble Tea TUI Dashboard) │
              └─────────────────────────────┘
```

### Network & Performance Constraints

- **Zero Memory Allocation for Large Payloads:** Stream HTTP request/response body data using `io.Copy` buffers to avoid reading massive binary blobs (like zip files or media) fully into heap memory.
- **Non-Blocking Logging:** Writing intercept history records to disk must never happen on the active socket's network goroutine loop. The proxy engine writes telemetry events to an in-memory ring-buffer channel, processed by an asynchronous SQLite worker pool.

## 2. Technical Component Breakdown

### A. The Interception & MitM Layer

- **The Engine Core:** Leverage `elazarl/goproxy` or a customized wrapper over `net/http/httputil.ReverseProxy`.
- **Dynamic Cert Generation:** On initial boot, `optic` mints a 4096-bit RSA self-signed Root CA Certificate (`ca.crt` and `ca.key`).
- **Runtime Snipping:** When a client establishes an outbound HTTPS handshake over an HTTP CONNECT tunnel, the proxy intercepts the host string, dynamically mints an X.509 leaf certificate scoped exactly to that hostname, signs it with the Root CA, and establishes a secure TLS loop back to the client.

### B. The Storage Engine

- **The Engine Database:** Embedded SQLite compiled without external CGO requirements (`modernc.org/sqlite` or `ncruces/go-sqlite3`).
- **Performance Optimization Flags:** Execute database handles with aggressive WAL (Write-Ahead Logging) optimization configurations to allow simultaneous read/write accessibility during active multithreaded intruder fuzzing tasks:
  ```sql
  PRAGMA journal_mode = WAL;
  PRAGMA synchronous = NORMAL;
  PRAGMA cache_size = -64000; -- Allocate up to 64MB memory caching pages
  ```

## 3. Core Feature Specification (The Modules)

### Module 1: The Interceptor (Live Traffic Breakpoint)
- **The Logic Pattern:** Introduce a thread-safe global execution barrier state.
- **Mechanism:** When the user flips "Intercept On", incoming requests fall into a blocked state via a conditional Go synchronization channel structure:
  ```go
  type InterceptBarrier struct {
      sync.Mutex
      Active bool
      Hold   chan bool
  }
  ```
- **TUI Presentation:** The blocked thread surfaces a visual overlay pane inside the interface workspace, rendering editable fields containing the raw HTTP text buffer. The user can rewrite any headers or request bodies manually before executing hotkeys for Forward or Drop.

### Module 2: The Repeater (Manual Crafting)
- **The Logic Pattern:** A clean split-window layout context.
- **Mechanism:** Pulls an arbitrary transaction historical payload from the SQLite database, loads the unencrypted text configuration directly into an interactive text editor panel model inside the TUI viewport, and triggers isolated outbound `http.Client` execution requests completely independent of the browser's session lifecycle.

### Module 3: The Intruder (Concurrent Wordlist Fuzzer)
- **The Logic Pattern:** Worker pool patterns (goroutine clusters consuming from a single workload channel).
- **Mechanism:** The user defines payload markers (e.g., `§fuzz_target§`) within a template request structure and mounts a local target wordlist file.
- **Execution Strategy:**
  1. A single thread stream reads line entries sequentially from the wordlist file.
  2. Strings are piped to a buffered job payload channel.
  3. An array of $N$ worker goroutines concurrently pulls items from the workload channel, interpolates the string substitution values into the target payload frame template, executes the request pipeline, and registers output statuses.

## 4. User Interface Model (Terminal UI Architecture)

Instead of complex web dashboard pipelines, map your interactive terminal layouts directly using Charmbracelet Bubble Tea (`charmbracelet/bubbletea`) to handle responsive state loop tracking.

### TUI Workspace Division Layout
Split the viewport surface into clean, functional regions using `lipgloss`:
- **The Master Index Panel (Top Half):** A rolling dynamic rows table matrix tracking live transaction entries: ID, Verb, Host, Path, Status Code, and Payload Size.
- **The Inspection Details View (Bottom Half - Left Panel):** The active editable text buffer tracking raw outbound HTTP requests.
- **The Inspection Details View (Bottom Half - Right Panel):** The corresponding read-only incoming HTTP server response payload view.
