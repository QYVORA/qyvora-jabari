# Validation

The validation stage is where JABARI refuses to guess. After analysis
produces findings, validation attempts to **confirm** each finding through
further, still non-destructive, queries.

## Why it exists

A rule may produce a finding from incomplete information. Validation gives
findings a truthful `Status`:

| Status | Meaning |
|---|---|
| `detected` | analysis flagged it; not yet confirmed |
| `confirmed` | validation re-checked and verified the condition |
| `false-positive` | validation proved the finding incorrect |

## What validation does

For each finding with `Status == detected`, the validation stage looks up a
confirmation handler keyed by rule ID. If one exists, it runs:

- additional `getprop` reads,
- a re-read of the relevant application metadata,
- a safe re-query that cannot change device state.

If a handler confirms, the finding becomes `confirmed`. If it contradicts,
the finding becomes `false-positive`. If no handler exists, the finding stays
`detected` with its original confidence — JABARI does not upgrade confidence
it cannot support.

## Design constraints

- **Never mutates.** Confirmation uses reads only.
- **Fail closed.** A failed check leaves the finding `detected`; it never
  downgrades to a silent pass or upgrades to confirmed on ambiguous output.
- **Deterministic.** Confirmation handlers are pure with respect to device
  state and logged input.

## Planned

- Per-rule confirmation modules (the interface exists; handlers are added
  alongside each rule).
- Optional opt-in checks that require higher privilege (e.g. temporary ADB
  root), always gated and reversible.
