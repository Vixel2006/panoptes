# penoptes

a high-performance, open-source intercepting proxy for the security community.

## why

because java is slow and existing proxies consume all your ram. penoptes is fast, lean, and terminal-native.

- **request groups** — start a group before a login flow, close it after, study every packet in isolation
- **notes** — attach notes to requests, groups, or hostnames; one place for all your endpoint research
- **repeater** — pull a captured request, modify headers/body, fire it off independently (burp-style)
- **interceptor** — live breakpoint: block threads, edit on the fly, forward or drop

zero memory allocation for massive payloads. non-blocking logging. go brr.

## install

```sh
git clone https://github.com/Vixel2006/penoptes.git
cd penoptes
go build -o penoptes ./cmd/penoptes
```

## usage

```sh
./penoptes
```

(flags coming. for now just run it.)

## architecture

```
                    [ Browser / Client ]
                             │
                             ▼
                ┌────────────────────────┐
                │    Proxy Engine        │  ← goproxy, MITM, cert mgmt
                │    (internal/adapters  │
                │     /outbound/proxy)   │
                └───────────┬────────────┘
                            │
                ┌───────────┴────────────┐
                │    Core Services       │  ← interceptor, repeater, groups, notes
                │    (internal/core/)    │
                └───────────┬────────────┘
                            │
                ┌───────────┴────────────┐
                │    Presentation        │  ← Bubble Tea TUI (coming)
                │    (internal/adapters  │
                │     /inbound/)         │
                └────────────────────────┘
```

## features (planned)

| module | what it does |
|---|---|
| **interceptor** | live traffic breakpoint. block, edit, forward, drop |
| **repeater** | modify captured requests and resend independently |
| **groups** | scope a trace (e.g. login flow), then study it in isolation |
| **notes** | annotate requests, groups, or hosts — unified research log |
| **intruder** | concurrent wordlist fuzzer (workers pull from a channel, go fast) |

## why not burpsuite

burpsuite is closed-source, costs a kidney every year, and portswigger knows. this is free, fast, and yours to hack on.

