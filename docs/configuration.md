# Configuration

Configuration is resolved in order (first match wins):

1. **Defaults** built into the code
2. **Config file** — `-c/--config`, `JABARI_CONFIG`, or auto-discovery
3. **Environment variables**
4. **CLI flags**

## Config file discovery

`jabari` looks for `config.yaml` (or `config.yml`, `config.json`) in:

- `$HOME/.jabari/`
- `$HOME/.config/jabari/`
- `$HOME/.config/androidsec/`
- the current directory

Set an explicit path with `-c` or `JABARI_CONFIG` to bypass discovery.

## Example

```yaml
profile: standard
output: terminal
log:
  level: info
authorized: false

timeout:
  device: 10s
  connect: 5s

report:
  dir: reports
  formats: [terminal, json, markdown]

enumeration:
  detail: full          # or: minimal
  include_system: false

validation:
  enabled: true

evidence:
  enabled: true
  dir: evidence
```

## Keys

| Key | Default | Meaning |
|---|---|---|
| `profile` | `standard` | pipeline profile |
| `output` | `terminal` | default report format |
| `log.level` | `info` | `debug`, `info`, `warn`, `error` |
| `authorized` | `false` | treat runs as pre-authorized |
| `timeout.device` | `10s` | per-adb-call timeout |
| `timeout.connect` | `5s` | transport connect timeout |
| `report.dir` | `reports` | session output directory |
| `enumeration.detail` | `full` | `full` or `minimal` |
| `enumeration.include_system` | `false` | include system packages |
| `validation.enabled` | `true` | run validation stage |
| `evidence.enabled` | `true` | capture evidence |
| `evidence.dir` | `evidence` | evidence output dir |

## Environment variables

| Variable | Maps to |
|---|---|
| `JABARI_CONFIG` | config file path |
| `JABARI_PROFILE` | `profile` |
| `JABARI_OUTPUT` | `output` |
| `JABARI_VERBOSE` | log level debug |
| `JABARI_QUIET` | suppress info output |
| `JABARI_AUTHORIZED` | `authorized` |

`ANDROIDSEC_*` variables are accepted as aliases.

## Flag precedence

Flags always win: `jabari assess usb --profile deep` overrides the config
file regardless of its `profile:` value.
