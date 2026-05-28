# Interception Adapter — Architecture

The interception adapter is the core of Panoptes. It sits between the client (browser) and the upstream server, decrypting, intercepting, and controlling every request/response.

---

## Entry Point: `HandleConn`

Every TCP connection accepted by `transport.Server` is handed to `HandleConn`. It creates a `bufio.Reader` and **peeks at the first byte** without consuming it, then routes to the correct handler:

| First byte | Meaning | Routed to |
|---|---|---|
| `0x16` | TLS ClientHello (SSL proxy or direct TLS) | `handleTLS` |
| `C` | Could be `CONNECT` | Peek 7 bytes → `handleConnect` or `handleHTTP` |
| Any other | HTTP method (`G`, `P`, `D`, `H`, ...) | `handleHTTP` |

The `bufio.Reader` is passed to all handlers so the peeked bytes are still available for `http.ReadRequest`.

```go
func (i *InterceptAdapter) HandleConn(conn net.Conn) {
    defer conn.Close()
    conn.SetDeadline(time.Now().Add(10 * time.Second))

    br := bufio.NewReader(conn)
    peek, err := br.Peek(1)
    if err != nil {
        return
    }

    switch {
    case peek[0] == 0x16:
        conn.SetDeadline(time.Time{})
        i.handleTLS(conn, br)
    default:
        conn.SetDeadline(time.Time{})
        fullPeek, err := br.Peek(7)
        if err != nil {
            return
        }
        if string(fullPeek) == "CONNECT" {
            i.handleConnect(conn, br)
        } else {
            i.handleHTTP(conn, br)
        }
    }
}
```

---

## Path A: `handleConnect` — HTTP CONNECT Tunnel (Explicit HTTPS Proxy)

Used when the browser is configured with **HTTP Proxy = localhost:8080** and accesses an HTTPS site.

### Flow

```
Browser                     Proxy                          Upstream
   │                          │                               │
   ├──CONNECT ex.com:443─────►│                               │
   │                          │  http.ReadRequest(br)         │
   │◄──200 Connection Est─────┤                               │
   │                          │                               │
   ├──[TLS ClientHello]──────►│                               │
   │     SNI: ex.com          │                               │
   │                          │  IssueLeaf("ex.com")          │
   │                          │  tls.Server(bc, config)       │
   │◄──[TLS ServerHello]──────┤                               │
   │◄──[TLS Cert: ex.com]─────┤                               │
   │                          │                               │
   │  ~~ TLS established ~~   │                               │
   │                          │                               │
   ├──GET / (encrypted)──────►│                               │
   │                          │  Decrypt → InterceptRequest   │
   │                          │  → barrier.Lock() → wait      │
   │                          │  → TUI prints → Release()     │
   │                          │  → RoundTrip over TLS         │
   │                          ├──────────GET /───────────────►│
   │                          │ ◄─────────response────────────┤
   │◄──response (encrypted)───┤                               │
```

### Implementation

```go
func (i *InterceptAdapter) handleConnect(conn net.Conn, br *bufio.Reader) {
    bc := &bufferedConn{Conn: conn, r: br}

    req, err := http.ReadRequest(br)
    if err != nil {
        return
    }
    host := req.Host

    conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

    tlsConfig := &tls.Config{
        GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
            name := hello.ServerName
            if name == "" {
                name, _, _ = net.SplitHostPort(host)
            }
            return i.certGen.IssueLeaf(name)
        },
    }
    tlsConn := tls.Server(bc, tlsConfig)
    if err := tlsConn.Handshake(); err != nil {
        return
    }

    i.interceptTLS(tlsConn, host)
}
```

### The `bufferedConn` Trick

After `http.ReadRequest(br)` consumes the CONNECT request, the browser may have already sent the TLS ClientHello (optimistic TLS pipelining). Those bytes sit in `br`'s internal buffer. When `tls.Server(bc, ...)` tries to read the ClientHello:

```
bc.Read() → br.Read() → returns buffered bytes first → then reads from raw TCP conn
```

No bytes are lost. The `bufferedConn` type:

```go
type bufferedConn struct {
    net.Conn
    r io.Reader
}

func (bc *bufferedConn) Read(b []byte) (int, error) { return bc.r.Read(b) }
```

---

## Path B: `handleTLS` — Direct TLS Connection (SSL/HTTPS Proxy)

Used when the browser is configured with **SSL Proxy = localhost:8080** (or the HTTPS proxy field). Firefox connects with TLS directly to the proxy. The proxy terminates the outer TLS, then reads HTTP from the decrypted stream.

### Flow

```
Browser                     Proxy                          Upstream
   │                          │
   ├──[TLS ClientHello]──────►│
   │     SNI: localhost       │
   │                          │  outer TLS handshake (MITM)
   │◄──[TLS established]──────┤
   │                          │
   │  Now reading HTTP from   │
   │  the decrypted TLS stream│
   │                          │
   ├──CONNECT ex.com:443─────►│  ← plaintext HTTP over outer TLS
   │                          │
   │  ┌────────────────────────────┐
   │  │  This is the SAME MITM     │
   │  │  flow as handleConnect,    │
   │  │  but over the outer TLS    │
   │  │  instead of raw TCP.       │
   │  └────────────────────────────┘
   │                          │
   │◄──200 Connection Est─────┤  ← over outer TLS
   │                          │
   ├──[Inner TLS ClientHello]─┤  ← through outer TLS tunnel
   │                          │  innerBC reads from outerBr
   │                          │  tls.Server(innerBC, ...)
   │◄──[Inner TLS established]┤
   │                          │
   │  ~~ Normal intercept ~~  │
   ├──GET / (inner encrypt)──►│
   │                          │  → interceptRequest → barrier
   │                          │  → RoundTrip over TLS
   │                          ├──────────GET /───────────────►
   │◄─────────response────────┤◄─────────response────────────┤
```

### Implementation

```go
func (i *InterceptAdapter) handleTLS(conn net.Conn, br *bufio.Reader) {
    bc := &bufferedConn{Conn: conn, r: br}

    outerConfig := &tls.Config{
        GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
            name := hello.ServerName
            if name == "" {
                name = "localhost"
            }
            return i.certGen.IssueLeaf(name)
        },
    }
    outer := tls.Server(bc, outerConfig)
    if err := outer.Handshake(); err != nil {
        return
    }
    defer outer.Close()

    outerBr := bufio.NewReader(outer)
    for {
        req, err := http.ReadRequest(outerBr)
        if err != nil {
            break
        }

        if req.Method == "CONNECT" {
            host := req.Host
            outer.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

            innerBC := &bufferedConn{Conn: outer, r: outerBr}
            innerConfig := &tls.Config{
                GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
                    name := hello.ServerName
                    if name == "" {
                        name, _, _ = net.SplitHostPort(host)
                    }
                    return i.certGen.IssueLeaf(name)
                },
            }
            inner := tls.Server(innerBC, innerConfig)
            if err := inner.Handshake(); err != nil {
                break
            }
            i.interceptTLS(inner, host)
            break
        }

        // Regular HTTP over outer TLS
        ...
        i.handleInterceptedHTTP(outer, req)
    }
}
```

### Nested TLS Detail

For the CONNECT-over-TLS case, the same `bufferedConn` pattern applies again:

```go
innerBC := &bufferedConn{Conn: outer, r: outerBr}
```

After `http.ReadRequest(outerBr)` reads the CONNECT from the outer TLS stream, `outerBr` may have the inner ClientHello buffered. `innerBC` serves those bytes first before reading from the outer TLS connection.

---

## Path C: `handleHTTP` — Plain HTTP

Simplest path. The HTTP request arrives raw (no TLS), is intercepted, and forwarded upstream over plain TCP.

```go
func (i *InterceptAdapter) handleHTTP(conn net.Conn, br *bufio.Reader) {
    req, err := http.ReadRequest(br)
    if err != nil {
        return
    }
    if !req.URL.IsAbs() {
        req.URL.Scheme = "http"
        req.URL.Host = req.Host
    }

    i.interceptor.InterceptRequest(req)
    i.barrier.Lock()
    forward := i.barrier.Decision()
    i.barrier.Unlock()

    if !forward {
        // Send 403 Forbidden
        ...
        return
    }

    transport := &http.Transport{Proxy: nil}
    upstreamResp, err := transport.RoundTrip(req)
    ...
    i.interceptor.InterceptResponse(upstreamResp)
    upstreamResp.Write(conn)
}
```

---

## Shared Loop: `interceptTLS`

Both `handleConnect` and `handleTLS` (for CONNECT) converge into the same intercept loop. This runs over an established TLS connection (either the MITM TLS from CONNECT or the inner TLS from nested TLS).

```go
func (i *InterceptAdapter) interceptTLS(tlsConn net.Conn, host string) {
    tlsBr := bufio.NewReader(tlsConn)
    transport := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
    }
    for {
        req, err := http.ReadRequest(tlsBr)
        if err != nil {
            break
        }
        req.URL = &url.URL{Scheme: "https", Host: host, Path: req.RequestURI}
        req.RequestURI = ""

        i.interceptor.InterceptRequest(req)
        i.barrier.Lock()
        forward := i.barrier.Decision()
        i.barrier.Unlock()

        if !forward {
            dropResp := &http.Response{
                StatusCode: http.StatusForbidden,
                ProtoMajor: 1, ProtoMinor: 1,
            }
            dropResp.Write(tlsConn)
            continue
        }

        upstreamResp, err := transport.RoundTrip(req)
        if err != nil {
            errResp := &http.Response{
                StatusCode: http.StatusBadGateway,
                ProtoMajor: 1, ProtoMinor: 1,
            }
            errResp.Write(tlsConn)
            continue
        }
        i.interceptor.InterceptResponse(upstreamResp)
        upstreamResp.Write(tlsConn)
        upstreamResp.Body.Close()
    }
}
```

The loop handles HTTP/1.1 keep-alive: multiple requests can flow over the same TLS connection.

---

## The Interceptor + Barrier (Request Control)

The `Interceptor` and `Barrier` together implement the "ack" mechanism — pausing and approving each request.

### `Interceptor.InterceptRequest`

```go
func (i *Interceptor) InterceptRequest(r *http.Request) error {
    rawBody, _ := io.ReadAll(r.Body)
    r.Body.Close()

    // Restore body so upstream RoundTrip can read it
    r.Body = io.NopCloser(bytes.NewReader(rawBody))
    r.ContentLength = int64(len(rawBody))
    r.Header.Del("Transfer-Encoding")

    // Snapshot to model (sent to TUI via channel)
    req := model.Request{ ... }
    select {
    case i.requestCh <- req:
    default: // drop if TUI is backed up
    }
    return nil
}
```

### `Barrier.Lock` / `Release`

```go
func (b *Barrier) Lock() {
    b.Mutex.Lock()
    if b.active {
        b.Mutex.Unlock()
        b.decision = <-b.hold          // BLOCK here
        b.Mutex.Lock()
    } else {
        b.decision = true              // auto-forward when inactive
    }
}

func (b *Barrier) Release(forward bool) {
    b.hold <- forward                  // unblock the proxy goroutine
}
```

The flow:
1. Proxy calls `InterceptRequest` → sends snapshot to TUI channel
2. Proxy calls `barrier.Lock()` → blocks on `<-b.hold`
3. TUI receives from request channel, prints request, calls `Release(true/false)`
4. Barrier sends decision to `b.hold` channel
5. Proxy wakes up, reads decision, forwards or drops

---

## Response Body Fix

After `transport.RoundTrip` returns, Go's HTTP client has already:
- Dechunked the body (if `Transfer-Encoding: chunked`)
- Read trailers, etc.

The raw response headers still contain `Transfer-Encoding: chunked`, but the body is now flat. If we blindly call `resp.Write(client)`, the browser gets:

```
HTTP/1.1 200 OK
Transfer-Encoding: chunked
                          ← body is NOT chunked →
```

The browser tries to dechunk the dechunked data → corruption → `PR_END_OF_FILE_ERROR`.

**Fix** in `InterceptResponse`:

```go
func (i *Interceptor) InterceptResponse(r *http.Response) error {
    rawBody, _ := io.ReadAll(r.Body)
    r.Body.Close()

    r.Body = io.NopCloser(bytes.NewReader(rawBody))
    r.ContentLength = int64(len(rawBody))
    r.Header.Del("Transfer-Encoding")

    // ... snapshot to model ...
    return nil
}
```

Same fix applies to `InterceptRequest` for outgoing bodies.

---

## Full Decision Tree

```
                   ┌──────────────────────────────────┐
                   │         HandleConn               │
                   │   bufio.Reader.Peek(1)           │
                   └──────┬──────────┬──────────┬─────┘
                          │          │          │
                    0x16  │    "C"   │   other  │
                          │          │          │
                    ┌─────▼──┐  ┌────▼───┐  ┌──▼──────┐
                    │handle  │  │Peek(7) │  │handle   │
                    │  TLS   │  │CONNECT?│  │  HTTP   │
                    └───┬────┘  └───┬────┘  └────┬────┘
                   TLS  │     yes   │   no       │
                   MITM │           │  ┌─────────┘
                    ┌───▼──┐  ┌────▼──▼─┐
                    │outer │  │handle   │
                    │ TLS  │  │Connect  │
                    └───┬──┘  └────┬────┘
                   CONNECT│   TLS  │
                    ┌────▼──┐  ┌───▼────┐
                    │inner  │  │  MITM  │
                    │ TLS   │  │  TLS   │
                    └────┬──┘  └───┬────┘
                         │         │
                    ┌────▼─────────▼────┐
                    │   interceptTLS    │
                    │  OR handleHTTP    │
                    │                   │
                    │  InterceptRequest │
                    │  → barrier.Lock() │
                    │  → TUI prints     │
                    │  → Release()      │
                    │  → RoundTrip()    │
                    │  → InterceptResp  │
                    │  → Write(client)  │
                    └───────────────────┘
```
