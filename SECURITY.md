# Security Policy

## Reporting a vulnerability

Please report security vulnerabilities **privately** — do not open a public
issue for them.

- **Contact:** create a private advisory in this repository (GitHub
  Security → Report a vulnerability), or email the maintainers per the
  address in `GOVERNANCE.md`.
- **What to include:**
  - affected version / commit,
  - description of the issue and its impact,
  - steps to reproduce,
  - any suggested mitigation.

## Supported versions

The latest release and the `main` branch are supported. Security fixes land
in the next release and are backported only to the latest release branch.

## Our commitment

- We will acknowledge your report within 3 business days.
- We will provide a status update within 10 business days.
- We will coordinate public disclosure after a fix is released, and credit
  reporters who opt in.

## Scope

In scope: vulnerabilities in JABARI itself — device-output parsing, command
construction, transport handling, evidence/session handling, authorization
logic, and the CLI.

Out of scope: vulnerabilities in the assessed Android device or apps; these
are findings, not framework vulnerabilities. See the
[Security Model](docs/Security-Model.md).
