# Installation

## Requirements

- **Go 1.21+** (build from source). The project's `go.mod` pins the
  toolchain.
- **Android platform-tools** (`adb`) on `PATH` for USB and network
  assessments.
- An **authorized** target device or emulator for live assessments. None is
  needed to build or test.

## Build

```sh
make build
```

This produces:

- `bin/jabari` — the primary binary
- `bin/androidsec` — symlink alias to the same binary

Or manually:

```sh
go build -o bin/jabari ./cmd/jabari
ln -sf jabari bin/androidsec   # optional
```

## Verify

```sh
./bin/jabari version
./bin/jabari --help
```

## Tooling

| Tool | Purpose |
|---|---|
| `adb` | device communication (USB + network transports) |
| `go test ./...` | unit tests (no hardware required) |
| `make test` / `make vet` | convenience wrappers |
| `make fmt` | `gofmt -w .` |

## Uninstall

Remove the `bin/` outputs:

```sh
make clean
```
