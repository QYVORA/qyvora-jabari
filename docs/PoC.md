# Proof-of-Concept Modules

JABARI's PoC subsystem (`internal/poc`) takes confirmed findings and tries to
prove them against the live, **authorized** target. A PoC module runs a
device command through the active transport, captures the command's output as
evidence, and records whether the finding was *proven* or *not proven*. Every
module is opt-in, non-destructive by default, and gated behind the same
authorization model as the rest of the pipeline.

## Lifecycle and state machine

Each PoC module is executed inside the `poc` stage and recorded as a
`models.PocRun` with one of four terminal states:

| Status | Meaning |
|---|---|
| `proven` | the module reproduced the finding and captured evidence |
| `not-proven` | the module ran but could not reproduce the finding |
| `skipped` | the module was eligible but not executed (e.g. high-risk without opt-in) |
| `error` | the module failed with a non-`ExitCode` transport/execution error |

The stage runs every registered module that has at least one *eligible*
finding (evidence whose category/package matches the module's match rules).
Running a module advances the corresponding finding's exploitability ladder:

```
proven     → exploited
not-proven → dynamic
```

The `poc` stage emits module lifecycle events on the events stream:

- `module.started` — one per module that is about to run
- `module.completed` — one per module, carrying the full `PocRun` payload
  (`id`, `module`, `finding_id`, `status`, `risk`, `summary`, `evidence`,
  `duration_ms`)

`PocRun` records are persisted on the session (`Session.Pocs`) and rendered
by every report format (terminal, JSON, markdown, HTML).

## Authorization and safety gates

The `poc` stage is gated like every interactive assessment:

1. **Target authorization.** The stage refuses to run and returns exit code
   `3` (authorization declined) unless the current target is authorized
   (`--authorized`/`-y`, `QYVORA_AUTHORIZED=true`, or interactive consent).
2. **Risk gating.** Modules are classified by risk:
   - `android.run_as_debuggable` (medium) — proves debug-app access by
     executing `run-as <package> id`.
   - `android.world_readable_data` (low) — checks world-readable app data via
     `ls -l /data/data/<package>`.
   - `android.exported_activity` (**high**) — launches an exported activity via
     `am start` and confirms the process appears. Launching an activity changes
     device state, so this module is **skipped by default** unless
     `poc.high_risk` is enabled in config or `--poc-high-risk` is passed.
3. **No device mutation without a transport.** APK-only targets have no
   transport; the stage is a no-op that records nothing.

There is no configuration path that runs PoC modules without authorization.

## Module contract

A PoC module implements:

```go
type Module interface {
    ID() string
    Metadata() ModuleMeta          // name, description, risk level
    Match(*models.Finding) bool    // is this finding eligible for this module?
    Run(ctx context.Context, env *Env, finding *models.Finding) (Result, error)
}
```

`internal/poc.Registry` holds the builtin set; `Registry.Get/List/ListIDs`
back the `poc` stage and the `--poc-module` filter. Modules execute device
commands with `env.Transport.Execute(models.Request{Command: "shell"})`, so
tests inject a fake transport and never need adb.

## Running PoCs

```sh
# Full assessment including the PoC stage (authorized, non-interactive)
jabari assess usb --authorized --poc

# Allow high-risk modules that change device state
jabari assess ip 192.168.1.50 -y --poc --poc-high-risk

# Standalone: run PoC modules against the current authorized target
jabari poc
jabari poc --poc-module android.world_readable_data

# Plan only, no execution
jabari assess usb --poc --dry-run
```

The standalone `jabari poc` command runs the `poc` stage with the current
target, the `--poc-high-risk` and `--poc-module` flags, and the events/report
flags of the normal CLI.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 2 | usage or target-selection error |
| 3 | authorization declined (poc stage / `poc` command) |
