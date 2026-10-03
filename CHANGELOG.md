# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- **Unified version system** — `internal/version` now carries the canonical
  framework identity (framework, version, commit, date, build user, Go
  version/arch/os) stamped via `-ldflags`, plus official QYVORA contact
  details. `jabari version` (CLI and console) renders the full block in
  terminal and machine formats.
- **Contact details** — the `version` command, README, and `SECURITY.md`
  surface official QYVORA contact: https://qyvora.org ·
  qyvorasec@gmail.com · Tamale, Ghana.
- **ANSI hygiene** — terminal colors are disabled when stdout is piped or
  redirected or `NO_COLOR` is set.

### Added


- Foundation release of the JABARI Android Security Assessment Framework
  - `assess` pipeline: discovery → enumeration → analysis → validation →
    risk → reporting
  - USB (ADB) and network target modes with a transport abstraction
  - Rule engine with builtin rules `AND-001 … AND-007`
  - Evidence store with SHA-256 hashing
  - Risk scoring (severity × confidence)
  - Reporting: terminal, JSON, Markdown, HTML
  - CLI: `assess`, `target`, `discover`, `enumerate`, `analyze`, `validate`,
    `report`, `version`, `completion`
  - Profiles: `quick`, `standard`, `deep`, `application`, `device`,
    `network`, `compliance`, `research`
  - Config file, environment variables, and flag precedence
  - Authorization gate with non-interactive mode
  - Unit tests (no hardware required)
  - Documentation and community files
