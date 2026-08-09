# Architecture

This document describes how JABARI is built and why. It targets contributors
and reviewers; operators should read the [Getting started](getting-started.md)
guide instead.

## Design goals

1. **Authorized-first.** An explicit authorization gate precedes every
   assessment. There is no configuration that enables unauthenticated
   broad scanning.
2. **Android-centric, not network-centric.** The framework assesses one
   authorized target at a time — either a USB-connected device or a single
   known IP address.
3. **Modular and testable.** Every pipeline stage is a small Go interface;
   every parser is decoupled from device access and unit-testable with
   synthetic fixtures.
4. **Honest status reporting.** No stage claims success it did not perform,
   and no rule claims a result it could not verify.
5. **Extensible.** New stages, transports, rules, and report formats are
   added by implementing an interface, not by editing core logic.

## Package layout

```
cmd/jabari/            executable entry point; calls cli.Execute()
internal/cli/          cobra command tree, authorization, wiring
internal/core/         pipeline contracts: Stage, Env
internal/config/       viper-based configuration
internal/discovery/    device metadata via props
internal/enumeration/  application inventory (pm/dumpsys)
internal/analysis/     rule evaluation loop
internal/validation/   non-destructive finding confirmation
internal/risk/         severity × confidence scoring
internal/evidence/     evidence store (hashing, storage)
internal/reporting/    renderers: terminal, json, markdown, html
internal/rules/        rule interface + registry + builtin AND set
internal/orchestration/ profile builder + pipeline runner
internal/transport/    Transport interface, USB + network impls, factory
pkg/adb/               thin, injectable adb binary wrapper
pkg/models/            shared data model (Target, Session, Finding, …)
pkg/utilities/         small shared string/parse helpers
```

### The core contract

`internal/core` defines the two contracts everything plugs into:

```go
type Stage interface {
    Name() string
    Run(ctx context.Context, env *Env) error
}

type Env struct {
    Target    *models.Target
    Session   *models.Session
    Transport transport.Transport
    Rules     *rules.Registry
    Evidence  *evidence.Store
    Log       *logger.Logger
    Config    *viper.Viper
    Apps      []models.Application
}
```

A stage reads from and writes to `Env`. Stages never reach out to the device
directly — they go through `Env.Transport`, which is how tests inject fakes.

## The pipeline

`internal/orchestration` builds a pipeline from the selected profile and runs
it:

```
discovery → enumeration → analysis → validation → risk → reporting
```

- Each stage is timed and recorded on the session.
- A stage error fails the session with a non-zero exit code and a recorded
  error; partial results already collected are still persisted.
- Context cancellation (SIGINT/SIGTERM) aborts cleanly at the next stage
  boundary.

### Profiles

Profiles are configurations of *which* stages run. The current set:

| Profile | Stages | Use |
|---|---|---|
| `quick` | discovery → analysis → risk → reporting | Fast posture read |
| `standard` | all six | Default full assessment |
| `deep` | all six, extended enumeration | Research-grade |
| `application` | discovery → enumeration → analysis → validation → risk → reporting | App-focused |
| `device` | discovery → analysis → risk → reporting | OS-focused |
| `network` | discovery → analysis → risk → reporting | Network-triggered |
| `compliance` | all six, report only | Audit output |
| `research` | all six, raw evidence | Full fidelity |

## Transports

`internal/transport` defines the boundary to the device:

```go
type Transport interface {
    Connect(ctx context.Context) error
    Disconnect() error
    Info(ctx context.Context) (*models.DeviceInfo, error)
    Execute(ctx context.Context, req models.Request) (models.Response, error)
    String() string
}
```

- `USBTransport` — targets a connected ADB device by serial.
- `NetworkTransport` — targets exactly one `host:port` (default ADB port
  5555) via `adb connect`.
- `transport.NewForTarget(target)` — the factory that picks an implementation
  from the target model.

Transports wrap the minimal `pkg/adb` package, which itself accepts an
injectable runner so tests never need a real `adb` binary.

## Rules and the analysis stage

`internal/analysis` walks `Env.Apps` (and the device), evaluates each rule in
`Env.Rules`, and appends fired findings to `Env.Session`. Rules are
independent implementations of:

```go
type Rule interface {
    ID() string
    Metadata() models.RuleMetadata
    Evaluate(ctx context.Context, ctx2 *EvaluationContext) ([]models.Finding, error)
}
```

The builtin set (`internal/rules/builtin`) currently provides `AND-001` …
`AND-007`. See [Rules](rules.md).

## Risk model

`internal/risk` combines a finding's severity and confidence into an overall
level:

```
Level 0: informational
Level 1: low
Level 2: medium
Level 3: high
Level 4: critical
```

Severity and confidence are independent, model-level concepts; see
[Models](models.md) and [Reporting](reporting.md).

## Reporting

`internal/reporting` renders a `Session` to terminal, JSON, Markdown, or
HTML. Renderers are functions of the session only — no device access — which
keeps them trivially testable and allows offline report generation from saved
sessions.

## Configuration

`internal/config` loads (first match wins):

1. defaults
2. config file (`-c/--config`, else the discovery list)
3. environment variables (`JABARI_*` / `ANDROIDSEC_*`)
4. CLI flags

See [Configuration](configuration.md).

## Authorized-only flow

The authorization gate (`internal/cli/authorization.go`) enforces:

- interactive confirmation on `assess`/`target` when the terminal is a TTY
- `-y/--yes` (or `--authorized`) for non-interactive runs
- refusal otherwise, with an explicit error and non-zero exit

There is no path that silently proceeds without authorization.
