# Targets

A **target** is the single Android device JABARI is authorized to assess. A
target can be selected two ways; a third (offline APK) is planned.

## Target modes

| Mode | Command | Transport | Scope |
|---|---|---|---|
| USB | `jabari assess usb` | USBTransport (ADB, by serial) | the connected device |
| Network | `jabari assess ip <addr>` | NetworkTransport (ADB over TCP, default 5555) | exactly that address |
| APK (planned) | `jabari assess apk <file>` | none (offline analysis) | the provided APK |

## USB targets

When you run `jabari assess usb`:

1. JABARI enumerates `adb devices -l`.
2. If exactly one device is listed, it is selected.
3. If several are listed, you are asked to choose (interactive) or pass a
   serial: `jabari assess usb <serial>`.
4. The authorization gate runs; then the pipeline begins.

Discovered metadata (from `getprop`):

- `ro.product.manufacturer`, `ro.product.model`, `ro.product.brand`
- `ro.build.version.release`, `ro.build.version.sdk`, `ro.build.version.security_patch`
- `ro.build.fingerprint`, `ro.build.version.codename`
- `ro.debuggable`, `ro.secure`, `ro.adb.secure`
- `ro.kernel.qemu` (emulator), `ro.boot.hardware`, `ro.product.board`
- kernel version (`uname -r`), build description
- best-effort rooted check (`su -c id` when permitted)

## Network targets

`jabari assess ip 192.168.1.50` connects via ADB over TCP to
`192.168.1.50:5555` (overridable with an explicit port). Requirements:

- The device has ADB-over-network enabled:
  ```sh
  adb tcpip 5555
  ```
- The address is reachable and the ADB authorization is accepted.
- **You are authorized to assess this device.**

JABARI does **not**:

- scan the surrounding subnet,
- ping-sweep neighbors,
- attempt other ports, or
- infer other hosts from the network.

The network transport connects to the supplied address and nothing else.

## Authorization

Both modes pass through the same gate:

- **TTY:** interactive `Are you sure you want to assess <target>? [y/N]`.
- **Non-interactive:** requires `-y/--yes`, config `authorized: true`, or
  `JABARI_AUTHORIZED=true`; otherwise the run aborts with a clear error.

The target's authorized flag is recorded on the session for the audit trail.

## Emulators

Emulators (including AVD) appear in `adb devices` normally and can be
assessed the same way as hardware. `ro.kernel.qemu=true` is surfaced in
device metadata.

## Future: offline APK targets

An `apk` target mode for static analysis is planned (see
[Roadmap](Roadmap.md)). It will have no live transport; analysis will be
purely offline against the APK file.
