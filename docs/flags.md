# cli flags

```
Usage: panoptes [flags]
```

## session selection

| flag | shorthand | description |
|------|-----------|-------------|
| `-session "name"` | `-s "name"` | start or resume a session by name. if it exists, load it. if not, create it. |

without either flag, panoptes opens the TUI with no active session. press `n` to create one or `s` to pick from the list.

## session listing

| flag | shorthand | description |
|------|-----------|-------------|
| `-list` | `-l` | list all available sessions and exit. useful for scripting and quick lookups. |

when used with `-list`, panoptes does not start the proxy or the TUI — it just prints sessions to stdout and exits.

## examples

```sh
# start with a new session named "bugbounty-target"
panoptes -session bugbounty-target

# same thing, shorthand
panoptes -s bugbounty-target

# resume an existing session
panoptes -s "my-session-name"

# list all saved sessions
panoptes -l

# pipe session list to grep
panoptes -l | grep target
```
