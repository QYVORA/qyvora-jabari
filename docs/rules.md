# Rules

JABARI's analysis stage evaluates security rules against the target device
and its applications. Rules are small, independent, and unit-tested.

## Rule interface

Rules implement the `rules.Rule` interface and are registered in the global
registry by the builtin package at init:

```go
type Rule interface {
    ID() string
    Metadata() models.RuleMetadata
    Evaluate(ctx context.Context, rctx *rules.EvaluationContext) ([]models.Finding, error)
}
```

`EvaluationContext` provides the rule with the current target, device info,
application list, and the evidence store — everything a rule needs to
produce a finding.

## Finding anatomy

Every rule produces zero or more `models.Finding`:

```go
type Finding struct {
    ID          string
    RuleID      string
    Title       string
    Description string
    Severity    models.Severity   // informational .. critical
    Confidence  models.Confidence // low .. high
    Status      models.FindingStatus // detected / confirmed / false-positive
    Evidence    []models.EvidenceRef
    DeviceInfo  models.DeviceInfo
    Details     map[string]string
}
```

## Builtin rules

The initial set (IDs `AND-001 … AND-007`) covers device posture:

| ID | Rule | Detection |
|---|---|---|
| `AND-001` | Debuggable production device | `ro.debuggable=1` on a non-userdebug, non-eng build |
| `AND-002` | Outdated security patch | security patch level older than a reference threshold |
| `AND-003` | Insecure USB connection | `ro.adb.secure=0` |
| `AND-004` | Rooted / userdebug build | `ro.debuggable` + root indicators |
| `AND-005` | User-visible build type | `ro.build.type` = `userdebug`/`eng` on a release device |
| `AND-006` | Emulator detected | `ro.kernel.qemu=1` (informational) |
| `AND-007` | ADB over TCP enabled | ADB network mode active (informational) |

Each rule:

- is **non-destructive** (reads properties only),
- records **evidence** for each finding,
- reports an honest `Confidence` (rules that cannot confirm set it low).

## Writing a rule

1. Add a file under `internal/rules/` implementing `Rule`.
2. Register it in `internal/rules/registry.go`.
3. Add table-driven tests in `internal/rules/`.
4. Document it in `docs/rules.md`.

See [Development](development.md) for conventions and test expectations.

## Rules and profiles

Profiles do not yet subset rules. All enabled rules run in every profile;
profile differences affect pipeline stages, not the rule set. Per-profile
rule filtering is planned.
