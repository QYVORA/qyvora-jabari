# Configuration

Configuration is resolved in order (first match wins):

1. **Defaults** built into the code
2. **Environment variables** (`QYVORA_*`)
3. **Config file** — `-c/--config`, or auto-discovery
4. **CLI flags** — flags always win

## Config file discovery

`jabari` looks for `config.yaml` (also `config.yml`, `config.json`) in:

- the current directory
- `$HOME/.qyvora-jabari/`
- `$HOME/.config/qyvora-jabari/`
- `$HOME/.config/qyvora/jabari/`
- `/etc/qyvora-jabari/`

Set an explicit path with `-c` (or a directory via `JABARI_CONFIG` is not
supported; use `-c path/to/config.yaml`) to bypass discovery. A missing file
is not an error; a malformed one is.

## Example

```yaml
profile: standard
output: terminal
verbose: false
quiet: false
json: false
authorized: false

log:
  level: info

report:
  dir: reports
  format: terminal   # terminal | json | markdown | html
```

## Keys

| Key | Default | Meaning |
|---|---|---|
| `profile` | `standard` | pipeline profile |
| `output` | `table` | default output format (`terminal`/`table`/`text`, `json`, `yaml`) |
| `verbose` | `false` | equivalent of `--verbose` |
| `quiet` | `false` | equivalent of `--quiet` |
| `json` | `false` | equivalent of `--json` |
| `authorized` | `false` | treat runs as pre-authorized |
| `log.level` | `info` | `debug`, `info`, `warn`, `error` |
| `report.dir` | `reports` | session output directory |
| `report.format` | `terminal` | default report format for `--report` |

Only keys actually read by the framework are listed; extra keys are ignored.

## Environment variables

Every key maps to an environment variable: uppercase, with dots and dashes
replaced by underscores, prefixed with `QYVORA_`. So `profile` →
`QYVORA_PROFILE`, `report.dir` → `QYVORA_REPORT_DIR`, and `log.level` →
`QYVORA_LOG_LEVEL`. `QYVORA_AUTHORIZED=true` is honored directly as well.

## Flag precedence

Flags always win: `jabari assess usb --profile deep` overrides the config
file and the `QYVORA_PROFILE` environment variable.
