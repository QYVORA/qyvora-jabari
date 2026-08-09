# Development

Guidance for contributors. See also [Architecture](architecture.md),
[Rules](rules.md), and the code of conduct.

## Setup

```sh
go test ./...
make vet
make fmt
```

Tests require **no hardware** — device access is behind interfaces that
tests fake.

## Layout conventions

- `internal/` — private implementation. Never import from outside the module.
- `pkg/` — shared/public packages (`models`, `adb`, `utilities`).
- `cmd/jabari/` — thin entry point; all wiring lives in `internal/cli`.

## Adding a pipeline stage

1. Define a struct implementing `core.Stage` (`Name()`, `Run(ctx, *core.Env)`).
2. Wire it into the pipeline in `internal/orchestration`.
3. Add a CLI command in `internal/cli` if it should be invocable standalone.
4. Add unit tests with a fake transport.
5. Document it in `docs/`.

A stage reads device data **only** through `Env.Transport` — never by calling
adb directly — so tests can substitute a fake.

## Adding a rule

See [Rules](rules.md). Minimum bar:

- table-driven tests with synthetic fixtures,
- honest confidence (do not claim high confidence on ambiguous input),
- evidence recorded for each finding,
- doc comment on the rule type,
- entry in `docs/rules.md`.

## Parser conventions

Parsers of device output (`dumpsys`, `pm list`, `getprop`) live in
`internal/enumeration` or `internal/discovery`. Rules:

- Parse from raw strings; never round-trip through another parser.
- Tolerate missing fields; return partial data, not errors, for malformed
  sections (log and continue).
- Return errors only for genuinely unrecoverable input.
- Cover synthetic fixtures in tests, including adversarial/malformed lines.

## Adding a transport

Implement `transport.Transport`. Register the target kind in the factory
`transport.NewForTarget`. Keep the minimal `pkg/adb` wrapper as the only
place that shells out to adb.

## Code style

- `gofmt` output; `go vet` clean; `go test ./...` green before pushing.
- Contexts flow from `main` down; stages respect cancellation.
- Errors wrap with context (`fmt.Errorf("…: %w", err)`).
- No dead code, no unused imports, no `panic` in library paths.

## Commit messages

Follow conventional commits:

```
feat: add network target mode
fix: parse dumpsys section header without panic
test: cover AND-001 debug build detection
docs: document evidence verification
```
