# penoptes Implementation Tasks

Here is the phase-based MVP delivery roadmap broken down into actionable tasks.

## Phase 1: Headless Forward Proxy
- [ ] Set up the basic Go project structure adhering to hexagonal architecture.
- [ ] Implement the core proxy engine using `elazarl/goproxy` or `net/http/httputil.ReverseProxy`.
- [ ] Ensure the proxy can forward unencrypted HTTP traffic.
- [ ] **Verification:** Deploy standalone binary, target external unencrypted servers using browser proxy configurations (`HTTP_PROXY=localhost:8080`).

## Phase 2: Dynamic TLS Mitigation Engine
- [ ] Implement RSA 4096-bit self-signed Root CA Certificate generation on boot (`ca.crt` and `ca.key`).
- [ ] Implement runtime snipping: intercept HTTPS handshake over HTTP CONNECT.
- [ ] Dynamically mint X.509 leaf certificates scoped to the requested hostname.
- [ ] Establish the secure TLS loop back to the client.
- [ ] **Verification:** Trust custom root certificates via local OS configuration setups, confirm clean decryption of outgoing HTTPS streams without cert failures.

## Phase 3: Asynchronous SQLite State Pipeline
- [ ] Integrate CGO-free embedded SQLite (`modernc.org/sqlite` or `ncruces/go-sqlite3`).
- [ ] Apply aggressive WAL optimization configurations.
- [ ] Implement the in-memory ring-buffer channel for telemetry events.
- [ ] Build the asynchronous SQLite worker pool to process writes from the channel.
- [ ] **Verification:** Execute parallel benchmark scripts, validating zero network request thread blockage during extreme write operations.

## Phase 4: The Interactive TUI Component Layout
- [ ] Initialize Charmbracelet Bubble Tea (`charmbracelet/bubbletea`) application state.
- [ ] Build the Master Index Panel (Top Half) with a dynamic rows table tracking live transactions.
- [ ] Build the Inspection Details View (Bottom Half - Left Panel) for editable raw outbound HTTP requests.
- [ ] Build the Inspection Details View (Bottom Half - Right Panel) for read-only incoming HTTP responses.
- [ ] Implement `InterceptBarrier` logic and wire up the Interceptor module visually.
- [ ] Implement the Repeater split-window layout and outbound execution logic.
- [ ] Implement the Intruder worker pool and wordlist fuzzing UI.
- [ ] **Verification:** Wire up Bubble Tea models, confirming clean UI tracking states, scrolling request text regions, and basic hotkey bindings.
