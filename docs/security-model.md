# Security Model

This document describes JABARI's trust boundaries, safety controls, and
operational guidance. It is required reading for contributors and for anyone
operating JABARI against devices.

## Authorized use only

JABARI is a tool for **authorized Android security assessment**. It is not a
network scanner, and it is not intended to access systems you do not own or
lack explicit permission to test. Operating it otherwise may violate law and
policy in your jurisdiction.

Every run passes an authorization gate before any device interaction:

1. The target is identified (USB device or specific IP).
2. The operator confirms authorization (interactively or via
   `-y`/config/env).
3. The session records that authorization.

## Trust boundaries

```
 Operator ── jabari (local process) ── adb ── Android target
                │
                └── reports/ sessions/ evidence/
```

- **JABARI runs locally** on the operator's machine. The target device is
  reached only through `adb`.
- **The target is untrusted input.** Device responses are parsed defensively;
  a hostile or malformed device response must not corrupt the session, crash
  the process, or leak host data. Parsers return only what they validate.
- **Evidence is captured locally** with SHA-256 hashing. Evidence files are
  written only to the configured output directory and recorded on the
  session.

## Safety controls

| Control | Detail |
|---|---|
| Authorization gate | every run must pass (see above) |
| No subnet scanning | network targets are exactly one address |
| Non-destructive by default | assessment stages run read-only queries |
| Validation is non-destructive | confirms findings by re-querying, never mutating |
| Timeouts | adb invocations run with timeouts from config |
| Cancellation | SIGINT/SIGTERM cancels the pipeline cleanly |
| Deterministic parsing | parsers fail closed, never panic on bad input |
| Evidence hashing | evidence artifacts are SHA-256 hashed at capture |

## What JABARI does not do

- It does not exploit, root, or install anything on a device in the
  foundation release. Exploitation validation is planned for a later phase
  and will be opt-in, versioned, and reversible.
- It does not persist anything on the target.
- It does not scan networks beyond the authorized target address.

## Logging and privacy

- Logs and reports may contain package names, version strings, and device
  metadata. Do not run JABARI against devices containing data you are not
  authorized to collect, and treat reports as sensitive artifacts.
- Session files and evidence are written with restrictive permissions where
  supported.

## Reporting security issues

See [SECURITY.md](../SECURITY.md). Do **not** open a public issue for
vulnerabilities.
