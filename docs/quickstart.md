# quickstart

five minutes. no java.

## install

```sh
git clone https://github.com/Vixel2006/penoptes.git
cd penoptes
go build -o panoptes ./cmd/panoptes
```

## set up your browser

panoptes generates its own root CA on first run so it can inspect HTTPS traffic. you need to trust it once.

```sh
# linux
sudo cp certs/panoptes-ca.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates

# macos
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain certs/panoptes-ca.crt

# windows
# why would you use window bro. just install linux or buy a macbook

# firefox (manual)
# Preferences → Privacy & Security → Certificates → View Certificates → Authorities → Import → pick certs/panoptes-ca.crt

```

now point your browser proxy to **localhost:8080**.

## first run

```sh
./panoptes
```

you'll see three panes:

- **sidebar** — sessions and groups
- **list** — captured requests
- **detail** — inspect the selected request/response/notes

the CA cert path is printed at startup. if you haven't installed it yet — now's the time.

## capture your first request

### 1. create a session

press **`n`**, type a name (e.g. "my-first-hunt"), hit enter. sessions isolate your work — different targets, different sessions.

### 2. create a group

press **`c`**, name it "login-flow" or whatever. groups let you scope a specific flow (e.g. the requests made during a login) and study them in isolation.

### 3. browse

head to any site in your browser. requests stream into the list pane in real time.

use **`j`** / **`k`** to navigate the list. the detail pane shows headers, body, and response for the selected request.

### 4. inspect

press **`1`** / **`2`** / **`3`** to switch between Request, Response, and Notes tabs in the detail pane.

use **`h`** / **`l`** or **left** / **right** to move focus between sidebar, list, and detail.

### 5. modify and resend

select a request and press **`e`** — the repeater opens. tweak the method, url, headers, or body, then hit Send. useful for testing parameter tampering or endpoint behaviour.

### 6. document with notes

press **`N`** to add a note to the active group. enter a title, then the body. use **`e`** to edit, **`d`** or **`x`** to delete. notes persist across sessions — one place for all your endpoint research.

### 7. filter

press **`/`** and type anything — url, method, status code, whatever. the list filters live as you type.

## features

| key | what it does |
|-----|-------------|
| **sessions** | press **`n`** to create, **`s`** to switch. sessions keep your work organised |
| **groups** | press **`c`** to create, **`g`** to pick one. assign requests with **`a`** (modal picker) or **`A`** (quick assign/unassign) |
| **notes** | press **`N`** to add, **`e`** to edit, **`d`** / **`x`** to delete. attached to the active group |
| **repeater** | press **`e`** on any request, modify, resend. burp-style, no kidney required |
| **interceptor** | press **`r`** to block traffic, **`R`** to let it through |
| **filter** | press **`/`** to search live |
| **pause** | press **`p`** to freeze the feed |
| **help** | press **`?`** to see all shortcuts in-app |

## hotkey cheat sheet

| key | action |
|-----|--------|
| `Tab` / `Shift+Tab` | cycle pane focus |
| `h` / `l` or `←` / `→` | move focus sidebar/list/detail |
| `j` / `k` or `↑` / `↓` | navigate / scroll |
| `1` / `2` / `3` | request / response / notes tab |
| `e` | open repeater (list) / edit note (notes tab) |
| `E` | configure preferred editor |
| `n` | new session |
| `s` | switch session |
| `c` | new group |
| `g` | select group |
| `a` | assign request to group (modal) |
| `A` | quick assign request to active group |
| `N` | new note |
| `d` / `x` | delete note |
| `/` | filter |
| `r` / `R` | interceptor on / off |
| `p` | pause / resume |
| `?` | help |
| `q` / `Ctrl+C` | quit |

## next steps

- read the [README](../README.md) for the big picture
- check [CONTRIBUTING.md](../CONTRIBUTING.md) if you want to hack on it
- `panoptes -l` lists all sessions from the command line
- `panoptes -s "name"` starts or resumes a session by name without the TUI menu
