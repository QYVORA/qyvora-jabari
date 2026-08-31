# JABARI — Android Security Assessment Framework

**Android Security Research, Assessment & Attack-Surface Framework**

JABARI is a modular Go framework for **authorized Android security
assessment**, attack-surface analysis, vulnerability validation, and
evidence-driven reporting across **USB-connected** and **specified-network**
Android targets.

> The CLI is `jabari`, published under the alias `androidsec`.

---

## What it is

A single interface for assessing an authorized Android device when you either:

- have the device connected over **USB** (via ADB), or
- know the device's **IP address** and are authorized to assess it.

JABARI is **Android-centric**, not a general network scanner. When a network
target is supplied, the framework assesses **that specific address only** — it
never auto-discovers or scans the surrounding subnet.

```
Discovery → Enumeration → Analysis → Validation → Evidence → Risk → Reporting
```

## What it is NOT

- Not a generic network scanner, Wi-Fi scanner, or mass IP scanner
- Not a malware or persistence-delivery framework
- Not an automated unauthorized-access tool

Every assessment requires **explicit target authorization**. All operations
are scoped, logged, and evidence-producing. JABARI is intended for use only
on systems you are authorized to test. See [SECURITY.md](SECURITY.md) and
[Security Model](docs/Security-Model.md).

## Status

**Foundation.** The core pipeline, data model, transports, rule engine,
evidence system, and reporting are implemented and tested. See
[Roadmap](docs/Roadmap.md) for what is planned (APK analysis, runtime
instrumentation, exploitation validation, ecosystem integration).

## Features

- **Two target modes** — USB (ADB) and known-IP network, with a transport
  abstraction so the pipeline is connection-agnostic
- **Assessment pipeline** — discovery, enumeration, analysis, validation,
  risk scoring, and reporting orchestrated by configurable profiles
- **Rule engine** — security checks are reusable, independently testable
  rules (initial set: `AND-001` … `AND-007`)
- **Evidence system** — every finding carries hashed, reproducible evidence
- **Risk scoring** — severity × confidence scoring per target
- **Reporting** — terminal, JSON, Markdown, and HTML
- **JSONL event stream** — `--events` emits a machine-readable run/stage/
  finding feed (stdout, stderr, or file) that agents and CI consume directly
- **Authorization gate** — explicit per-target confirmation, `--dry-run`
  friendly, non-interactive mode for automation
- **Profiles** — `quick`, `standard`, `deep`, `application`, `device`,
  `network`, `compliance`, `research`

## Install

The quickest way is the zero-config installer — it detects your platform
and installs the checksum-verified prebuilt binary plus the jabari icon and
desktop entry:

```sh
curl -fsSL https://raw.githubusercontent.com/QYVORA/qyvora-jabari/main/install.sh | bash
```

On Windows, use the PowerShell installer — it downloads the checksum-verified
binary under `%LOCALAPPDATA%\Programs\jabari\bin`, adds it to your PATH, and
installs the jabari icon with a Start Menu shortcut:

```powershell
irm https://raw.githubusercontent.com/QYVORA/qyvora-jabari/main/install.ps1 | iex
```

Building from source requires Go 1.21+ (project targets the toolchain in
`go.mod`) and, for USB or ADB-over-network targets, the Android platform-tools
(`adb`).

```sh
make build          # builds bin/jabari and the bin/androidsec alias
# or
go build -o bin/jabari ./cmd/jabari
```

Install `jabari` into your system so it appears in your commands and app
menu with the logo (see [Installation](docs/Installation.md)):

```sh
sudo make install          # system-wide (Linux/Unix), or:
make install-user          # per-user, no root
```

## Updating

```sh
jabari updates             # `jabari update` works as an alias; also available in the console
```

Checks the installed version against the latest official QYVORA GitHub
release, downloads the artifact for your platform, verifies its SHA-256
against the published `checksums.txt`, and swaps the binary in atomically.
Downgrades are refused, permission problems are reported clearly instead of
escalating on their own, and any failure leaves your current binary untouched —
no Go toolchain or Git required.

## Quick start

```sh
# Drop into the interactive console (Metasploit-style REPL)
jabari

# Assess a connected device (interactive authorization)
jabari assess usb

# Assess a specific authorized device by IP
jabari assess ip 192.168.1.50

# Non-interactive (automation), deep profile, JSON report
jabari assess ip 192.168.1.50 -y --profile deep --json
```

Inside the console you get a colored prompt, a live target/profile status
strip, tab completion, and history:

```text
jabariλ > target usb
jabariλ > assess
jabariλ > help
jabariλ > quit
```

Example terminal output:

```
ANDROIDSEC
──────────────────────────────────────────────

Session
  ID:      sess-9f2c1e4b7a
  Target:  tgt-3a8d5b2c01
  Profile: standard

Findings
  Critical       0
  High           1
  Medium         2
  Low            4
  Informational  7
  Total          14

Key findings
  [HIGH] Debuggable Production Device (confirmed)
```

## Documentation

| Document | Purpose |
|---|---|
| [Architecture](docs/Architecture.md) | Package layout and pipeline design |
| [Installation](docs/Installation.md) | Building from source |
| [Getting started](docs/Getting-Started.md) | First assessment |
| [CLI reference](docs/CLI.md) | Every command and flag |
| [Targets](docs/Targets.md) | USB and network target modes |
| [Configuration](docs/Configuration.md) | Config file and environment |
| [Rules](docs/Rules.md) | Rule engine and the AND-xxx rule set |
| [PoC modules](docs/PoC.md) | Proof-of-concept module system, safety, and events |
| [Validation](docs/Validation.md) | How findings are confirmed |
| [Evidence](docs/Evidence.md) | Evidence collection and storage |
| [Reporting](docs/Reporting.md) | Report formats |
| [Security model](docs/Security-Model.md) | Trust boundaries and safety controls |
| [Development](docs/Development.md) | Contributing code |
| [Roadmap](docs/Roadmap.md) | Planned work |

## Project layout

```
cmd/jabari/            entry point
internal/cli/          cobra command tree
internal/core/         pipeline stage contracts
internal/config/       configuration loading
internal/discovery/    device metadata
internal/enumeration/  application inventory
internal/analysis/     rule evaluation
internal/validation/   non-destructive confirmation
internal/risk/         severity/confidence scoring
internal/reporting/    terminal/json/markdown/html
internal/rules/        rule engine + builtin AND rules
internal/evidence/     evidence store
internal/orchestration/ pipeline + profiles
internal/transport/    USB/network transport abstraction
pkg/adb/               minimal adb wrapper
pkg/models/            shared data model
pkg/utilities/         small shared helpers
```

## License

Apache License 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and
[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Report security issues per
[SECURITY.md](SECURITY.md).
