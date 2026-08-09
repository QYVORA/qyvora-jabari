# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

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
