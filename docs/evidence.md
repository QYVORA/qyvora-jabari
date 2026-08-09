# Evidence

Every finding in JABARI can reference captured evidence. The evidence system
makes reports reproducible and findings auditable.

## How evidence is captured

During the pipeline, stages and rules can attach evidence to a finding. Each
evidence record captures the raw data behind the claim:

- `Type` — e.g. `shell-output`, `property`, `package-detail`
- `Source` — where it came from (e.g. `getprop ro.debuggable`)
- `Data` — the raw value
- `SHA256` — hash of the data at capture time
- `Timestamp` — capture time

Evidence is recorded on the finding and stored in the session, which is
serialized to the session JSON.

## Evidence store

`internal/evidence` owns the on-disk store:

- evidence artifacts are written under the configured `evidence.dir`
  (default `evidence/`)
- artifacts are SHA-256 hashed at write time
- the hash is recorded in the finding's `EvidenceRef`, linking report to file

## Integrity

- The session JSON records each evidence artifact's hash.
- A report rendered from a session references the hash, so a tampered
  artifact is detectable by re-hashing.
- Evidence is captured locally; the target never receives evidence files.

## Configuration

```yaml
evidence:
  enabled: true
  dir: evidence
```

Set `enabled: false` to skip on-disk evidence while still recording
in-memory evidence in the session (useful for read-only review workflows).

## Planned

- `jabari evidence verify <session>` to re-hash and report mismatches.
- Evidence redaction rules for sensitive fields before report export.
