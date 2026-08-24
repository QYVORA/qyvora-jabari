# Roadmap

Honest status of what is done, what is next, and what is deferred.

## Done (foundation)

- [x] Modular Go project layout (`cmd/`, `internal/`, `pkg/`)
- [x] Data model (`pkg/models`): targets, sessions, findings, evidence
- [x] Transport abstraction + USB and network implementations
- [x] Pipeline: discovery → enumeration → analysis → validation → risk → reporting
- [x] Profiles (`quick`, `standard`, `deep`, `application`, `device`, `network`, `compliance`, `research`)
- [x] Rule engine + builtin `AND-001 … AND-007`
- [x] Evidence store with SHA-256 hashing
- [x] Risk scoring (severity × confidence)
- [x] Reporting: terminal, JSON, Markdown, HTML
- [x] CLI: `assess`, `target`, `discover`, `enumerate`, `analyze`, `validate`, `report`, `version`, `completion`
- [x] Authorization gate, dry-run support, non-interactive mode
- [x] Config file + environment + flags
- [x] Unit tests covering parsers, rules, risk, models, reporting

## Next (v0.2: Static analysis)

- [ ] `apk` target mode: `jabari assess apk <file>` (offline)
- [ ] Manifest parsing, component enumeration, exported-component analysis
- [ ] Signing certificate inspection, permissions analysis
- [ ] Rule expansion for static findings

## Next (v0.3: Network & USB deep-dive)

- [ ] USB enumeration detail (USB device tree, transport info)
- [ ] Network hardening checks (reachable ports beyond ADB, exposed services)
- [ ] Wi-Fi related posture checks where the device exposes them
- [ ] Extended ADB diagnostics (SELinux, encryption, userdebug builds)

## Next (v0.4: Validation & exploitation-validation, opt-in)

- [ ] Non-destructive exploitation *validation*: confirm a finding's impact
      without harm, with explicit revert and audit trail
- [ ] Backup extraction check (read-only)
- [ ] Intent/protocol fuzzing against exported components (scoped, opt-in)
- [ ] CVE/known-vuln mapping

## Deferred / explicit non-goals

- [ ] Broad network scanning — **out of scope by design**
- [ ] Malware persistence or delivery tooling — **out of scope**
- [ ] Unauthorized-access automation — **out of scope**

## Guiding principle

> Every capability is gated, scoped, reversible where it could have impact,
> and clearly documented. Features are shipped only when they are honest
> about what they do.
