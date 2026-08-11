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

## Install into your environment

`jabari` ships with the tool's logo and a desktop entry. Installing it makes
the `jabari` command available, adds it to your application menu with the
logo, and is fully searchable — for example, typing `jabari` in the Kali
application launcher shows the tool with its icon.

### System-wide (Linux/Unix, typically requires root)

```sh
sudo make install
```

This installs:

- `/usr/local/bin/jabari` — the command
- `/usr/local/bin/androidsec` — alias symlink
- `/usr/local/share/applications/jabari.desktop` — desktop entry
- `/usr/local/share/icons/hicolor/512x512/apps/jabari.png` — icon
- `/usr/local/share/pixmaps/jabari.png` — icon (pixmap lookup)

`PREFIX` and `DESTDIR` are honored for packaging and staging:

```sh
make install PREFIX=/usr
make install DESTDIR=/tmp/pkgroot
```

### Per-user (no root)

```sh
make install-user
```

Installs the same layout under `~/.local` (`bin`, `share/applications`,
`share/icons`, `share/pixmaps`). If `~/.local/bin` is not on your `PATH`,
add it:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

### After installing

The binary and alias resolve on `PATH`:

```sh
which jabari androidsec
jabari version
```

The desktop entry is indexed by the launcher, so searching `jabari` in the
application menu shows the tool with its logo. Desktop and icon caches are
refreshed automatically when the cache tools are present.

## Uninstall

Remove installed files (system or user):

```sh
sudo make uninstall     # removes the system-wide install
make uninstall-user     # removes the per-user install
make clean              # removes local bin/ build outputs
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
| `make install` / `make install-user` | install into the environment with icon + desktop entry |

## Uninstall

Remove the `bin/` outputs:

```sh
make clean
```
