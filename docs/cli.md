# CLI Reference

`jabari` (alias `androidsec`) — command reference.

## Global flags

```
-c, --config string   config file (searched: ./config.yaml, ~/.qyvora-jabari/,
                      ~/.config/qyvora-jabari/, ~/.config/qyvora/jabari/, /etc)
-v, --verbose         verbose output
-q, --quiet           suppress informational output
    --json            output as JSON (shorthand for -o json)
-o, --output string   output format: terminal, json, markdown, html, yaml
    --events string   emit JSONL event stream to stdout, stderr, or a file path
-t, --timeout string  default timeout for device operations (default 30s)
    --dry-run         validate target + print the assessment plan, no execution
-y, --yes             skip interactive authorization confirmation
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
jabari poc                    run PoC modules against the current target
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
jabari > poc --poc-high-risk
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
- `--poc` — append the proof-of-concept stage to the pipeline (see
  [PoC Modules](poc.md))
- `--poc-high-risk` — allow PoC modules that change device state (e.g.
  `android.exported_activity`)
- `--json` / `-o <fmt>` — report format
- `--dry-run` — validate the target, the authorization decision and the
  profile, then print the stage plan without connecting to the device

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

## `poc`

Runs proof-of-concept modules against the current target and records the
results (`proven` / `not-proven` / `skipped` / `error`) on the session.
Refuses to run unless the target is authorized (exit code `3`).

```sh
jabari poc                          # run all eligible modules (authorized target)
jabari poc --poc-high-risk          # also allow state-changing modules
jabari poc --poc-module android.world_readable_data
```

See [PoC Modules](poc.md) for the module catalog, safety gates and event
schema.

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

The environment namespace is `QYVORA_*`. Every config key maps to an
environment variable of the same name, uppercased with dots replaced by
underscores (viper automatic binding), so `profile`, `output`, `report.dir`,
`log.level` become `QYVORA_PROFILE`, `QYVORA_OUTPUT`, `QYVORA_REPORT_DIR`,
`QYVORA_LOG_LEVEL` and so on. `QYVORA_AUTHORIZED=true` is also honored
directly. CLI flags take precedence over the environment.

## Events stream

`--events` emits one self-describing JSONL event per line to `stdout`,
`stderr`, or a file path (created/truncated mode 0600). Events carry
`schema_version`, `timestamp`, `execution_id`, `framework`, `level`, `event`
and `data`, covering the run lifecycle (`run.started`, `run.completed`),
pipeline stages, findings, warnings and errors. PoC modules additionally emit
`module.started` / `module.completed` lifecycle events whose `data` carries
the full `PocRun`. The schema is identical across the QYVORA frameworks.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | runtime, target/transport, or assessment error |
| 2 | usage, authorization, configuration, or target-selection error |
| 3 | authorization declined (PoC stage) |
