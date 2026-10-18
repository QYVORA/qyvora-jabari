# CLI Reference

`jabari` follows the shared QYVORA tier grammar:

```
jabari <tier> <module> [<action>] [target] [flags]
```

`<tier>` is one of `recon`, `exploit`, or `report`. Every pre-tier command remains available as a hidden alias.

## recon

| Command | Purpose |
|---------|---------|
| `jabari recon analyze` | Evaluate rules against the current target |
| `jabari recon assess` | Run a full assessment pipeline |
| `jabari recon capabilities` | Show native capabilities and optional integrations |
| `jabari recon completion` | Generate shell completion scripts |
| `jabari recon console` | open the legacy Metasploit-style device console |
| `jabari recon discover` | Identify the current target |
| `jabari recon enumerate` | Inventory applications on the current target |
| `jabari recon help` | Help about any command |
| `jabari recon poc` | Run proof-of-concept modules against the current target |
| `jabari recon report` | Render an assessment report |
| `jabari recon target` | Manage assessment targets |
| `jabari recon tools` | Report the Android assessment toolchain |
| `jabari recon tui` | start the interactive terminal application |
| `jabari recon updates` | Update jabari from official QYVORA GitHub releases |
| `jabari recon validate` | Confirm detected findings on the current target |
| `jabari recon version` | Print version information |

## exploit

| Command | Purpose |
|---------|---------|
| `jabari exploit analyze` | Evaluate rules against the current target |
| `jabari exploit assess` | Run a full assessment pipeline |
| `jabari exploit capabilities` | Show native capabilities and optional integrations |
| `jabari exploit completion` | Generate shell completion scripts |
| `jabari exploit console` | open the legacy Metasploit-style device console |
| `jabari exploit discover` | Identify the current target |
| `jabari exploit enumerate` | Inventory applications on the current target |
| `jabari exploit help` | Help about any command |
| `jabari exploit poc` | Run proof-of-concept modules against the current target |
| `jabari exploit report` | Render an assessment report |
| `jabari exploit target` | Manage assessment targets |
| `jabari exploit tools` | Report the Android assessment toolchain |
| `jabari exploit tui` | start the interactive terminal application |
| `jabari exploit updates` | Update jabari from official QYVORA GitHub releases |
| `jabari exploit validate` | Confirm detected findings on the current target |
| `jabari exploit version` | Print version information |

## report

_No report-tier modules._

## Utilities

| Command | Purpose |
|---------|---------|
| `jabari analyze` | Evaluate rules against the current target |
| `jabari assess` | Run a full assessment pipeline |
| `jabari console` | open the legacy Metasploit-style device console |
| `jabari discover` | Identify the current target |
| `jabari enumerate` | Inventory applications on the current target |
| `jabari poc` | Run proof-of-concept modules against the current target |
| `jabari target` | Manage assessment targets |
| `jabari tools` | Report the Android assessment toolchain |
| `jabari tui` | start the interactive terminal application |
| `jabari validate` | Confirm detected findings on the current target |
| `docs script <file>` | Run a `.qs` script with the shared scripting engine (`run`/`build` subcommands) |
