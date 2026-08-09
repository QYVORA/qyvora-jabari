# Reporting

JABARI renders every assessment session as a report. Reports are pure
functions of the saved session, which means they can be produced offline, in
any format, at any time.

## Session files

Each assessment produces a session JSON under `report.dir` (default
`reports/`):

```
reports/session-<id>.json
```

The session contains the target, device info, findings, evidence references,
stage timings, and profile. It is the single source of truth for reports.

## Formats

| Format | Flag | Use |
|---|---|---|
| terminal | `-o terminal` | human summary on stdout |
| text | `-o text` | plain-text report |
| json | `-o json` / `--json` | machine-readable |
| yaml | `-o yaml` | structured, human-readable |
| markdown | `jabari report -f markdown` | docs/sharing |
| html | `jabari report -f html` | self-contained page |

> `report -f`/`--format` selects the report format for rendering saved
> sessions; `-o`/`--output` selects the output format for `assess`.
> Both accept the same set of values.

## Rendering saved sessions

```sh
jabari report                 # latest session, terminal
jabari report --list          # list saved sessions
jabari report sess-abc123     # specific session
jabari report sess-abc123 -f html
```

## Report contents

- Session and target identifiers, profile, timestamps
- Device metadata (manufacturer, model, OS, patch level, build)
- Finding counts by severity
- Each finding: rule ID, title, severity, confidence, status, evidence refs
- Stage timings and any recorded errors

## Planned

- Summary and per-finding export to PDF and CSV
- Diff reports between sessions
- Configurable report templates
