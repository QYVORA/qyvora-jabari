# CLI Reference

`jabari` (alias `androidsec`) — command reference.

## Global flags

```
-c, --config string   config file (default: $HOME/.jabari/config.yaml)
-y, --yes             skip interactive authorization confirmation
-q, --quiet           suppress informational output
    --json            output as JSON (shorthand for -o json)
-o, --output string   report format: table, json, text, yaml
-h, --help            help
    --version         version
```

## Command summary

```
jabari assess usb|ip <addr>   run the full assessment pipeline
jabari target                 select/inspect the current target
jabari discover               run the discovery stage
jabari enumerate              run the enumeration stage
jabari analyze                run the analysis stage
jabari validate               run the validation stage
jabari report                 render a saved session
jabari version                print version
jabari completion <shell>     generate completion scripts
```

## Interactive console

Running `jabari` with no subcommand drops into a Metasploit-style console.
Every command above (and more) works as a console command, so you can
sequence a workflow without repeating the `jabari` prefix:

```
jabari > target usb
jabari > assess
jabari > report list
jabari > help
jabari > quit
```

Console extras: `help` / `?`, `banner`, `clear`, `history`, and
`set <key> <value>` / `get <key>` for live config overrides (e.g.
`set profile deep`, `set timeout 60s`, `set report.format json`).
Tab completion, command history, and a live target/profile status strip are
enabled on interactive terminals. Run `help` inside the console for the full
command list.

## `assess`

Runs the full pipeline and renders the report.

```sh
jabari assess usb                    # connected device
jabari assess usb <serial>           # specific connected device
jabari assess ip 192.168.1.50        # authorized device by IP
jabari assess ip 192.168.1.50:5555   # explicit port
```

Flags:

- `--profile <name>` — `quick|standard|deep|application|device|network|compliance|research` (default `standard`)
- `-y, --yes` — non-interactive authorization
- `--json` / `-o <fmt>` — report format

Exit codes: `0` success, `1` runtime/assessment error, `2` usage or
authorization error.

## `target`

```sh
jabari target                # show current target
jabari target usb            # select connected device (interactive if >1)
jabari target usb <serial>
jabari target ip <addr>      # select network target
```

## `discover`

Collects device metadata into the current session.

## `enumerate`

Inventories installed packages and their metadata.

## `analyze`

Evaluates all enabled rules against the current target and apps.

## `validate`

Re-runs non-destructive confirmation checks for each open finding and updates
their `confirmed` status.

## `report`

```sh
jabari report                 # render the most recent session
jabari report --list          # list saved sessions
jabari report <session-id>    # render a specific session
jabari report <session-id> -f html   # specific format
```

## `version`

Prints the build version, commit, date, and user. Add `--json` for machine
output.

## `completion`

```sh
jabari completion bash
jabari completion zsh
jabari completion fish
jabari completion powershell
```

## Environment variables

| Variable | Purpose |
|---|---|
| `JABARI_CONFIG` / `ANDROIDSEC_CONFIG` | config file path |
| `JABARI_PROFILE` | default profile |
| `JABARI_OUTPUT` | default output format |
| `JABARI_VERBOSE` | enable verbose logging |
| `JABARI_AUTHORIZED` | treat runs as authorized (`true`) |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | runtime, target/transport, or assessment error |
| 2 | usage, authorization, configuration, or target-selection error |
