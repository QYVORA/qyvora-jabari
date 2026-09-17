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
| `transport.native` | `false` | network targets use the native ADB-protocol transport (no `adb` binary dependency) |

Only keys actually read by the framework are listed; extra keys are ignored.

### Native transport

`transport.native` (`QYVORA_TRANSPORT_NATIVE=true`) switches network targets
(`adb connect` style, `host:port`) to Jabari's own implementation of the ADB
wire protocol instead of shelling out to the `adb` binary. This is useful in
locked-down environments where the Android SDK Platform Tools are not
installed.

Constraints:

- **Network targets only.** The flag is honored only for network targets; USB
  and APK targets keep their existing transports (legacy USB via the `adb`
  binary, static-only APK analysis). The native USB path requires libusb and
  is **not implemented**.
- **Default port.** A `host`-only address assumes the standard ADB port
  `5555`; a full `host:port` address is honored as given.
- **Scope is unchanged.** The authorization gate (`--authorized`, `-y`, or
  `QYVORA_AUTHORIZED=true`) applies identically; transport choice never
  bypasses authorization.

## Environment variables

Every key maps to an environment variable: uppercase, with dots and dashes
replaced by underscores, prefixed with `QYVORA_`. So `profile` →
`QYVORA_PROFILE`, `report.dir` → `QYVORA_REPORT_DIR`, and `log.level` →
`QYVORA_LOG_LEVEL`. `QYVORA_AUTHORIZED=true` is honored directly as well.

## Flag precedence

Flags always win: `jabari assess usb --profile deep` overrides the config
file and the `QYVORA_PROFILE` environment variable.
