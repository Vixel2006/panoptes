# panoptes

a high-performance, open-source intercepting proxy for the security community.

because java is slow and existing proxies consume all your ram. penoptes is fast, lean, and terminal-native.

- **request groups** — start a group before a login flow, close it after, study every packet in isolation
- **notes** — attach notes to requests, groups, or hostnames; one place for all your endpoint research
- **repeater** — pull a captured request, modify headers/body, fire it off independently (burp-style, no kidney required)
- **interceptor** — live breakpoint: block threads, edit on the fly, forward or drop

zero memory allocation for massive payloads. non-blocking logging. go brr.

## quickstart

```sh
git clone https://github.com/Vixel2006/penoptes.git
cd penoptes
go build -o panoptes ./cmd/panoptes
./panoptes
```

see [docs/quickstart.md](docs/quickstart.md) for the full 5-minute walkthrough — sessions, groups, notes, repeater, filters, the works.

## architecture

penoptes uses a clean **hexagonal architecture** (ports and adapters). the core proxy engine knows nothing about the TUI or SQLite — it just intercepts traffic and pushes data to channels.

```
                    [ Browser / Client ]
                             │
                             ▼
                 ┌────────────────────────┐
                 │   InterceptAdapter     │  ← MITM, TLS, HTTP engine
                 │   (internal/adapters)  │
                 └───────────┬────────────┘
                             │
                 ┌───────────┴────────────┐
                 │   Application Layer    │  ← interceptor, session/group/note managers
                 │   (internal/app/)      │
                 └───────────┬────────────┘
                             │
                 ┌───────────┴────────────┐
                 │    Core Domain         │  ← pure models + port interfaces
                 │    (internal/core/)    │
                 └───────────┬────────────┘
                             │
                 ┌───────────┴────────────┐
                 │    Presentation        │  ← Bubble Tea TUI
                 │    (internal/ui/)      │
                 └────────────────────────┘
```

## features

| module | what it does |
|--------|-------------|
| **interceptor** | live traffic breakpoint. block, edit, forward, drop |
| **repeater** | modify captured requests and resend independently |
| **groups** | scope a trace (e.g. login flow), then study it in isolation |
| **notes** | annotate requests, groups, or hosts — unified research log |
| **sessions** | organise your work into named sessions, switch between them |
| **filter** | search captured traffic by url, method, status code, body text |
| **intruder** (planned) | concurrent wordlist fuzzer (workers pull from a channel, go fast) |

## contributing

see [CONTRIBUTING.md](CONTRIBUTING.md). don't make a mess.

## why not burpsuite

burpsuite is closed-source, costs a kidney every year, and portswigger knows. this is free, fast, and yours to hack on.
