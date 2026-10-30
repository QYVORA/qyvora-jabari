# qyvora-jabari — Capability and Domain Audit

**Status:** `AUDIT COMPLETE` · **Audit date:** 2026-10-21 · **Mode:** audit only, no implementation
**Governing prompt:** `products/QYVORA-AI/PROMPT.md` (§5 Android domain, §6 technique classification, §13 Deliverable A)

---

## 1. Framework identity and intended domain

| Field | Value | Evidence |
|---|---|---|
| Name | `jabari` | `internal/version`, `bin/jabari version -o json` |
| Module | `github.com/QYVORA/qyvora-jabari` | `go.mod:1` |
| Go directive | `go 1.26.5` | `go.mod:3` |
| Version / commit | `0.1.0` / `847bded` | `bin/jabari version -o json` |
| Build | `go1.27.1`, linux/amd64, `2026-10-21T01:21:57Z` | same |
| Domain | Authorized **Android security assessment**: USB/ADB and specified-IP targets, plus offline APK static analysis | `internal/cli/root.go` appDescription; `README.md:14-28` |
| Intended non-goals | Broad network scanning, malware delivery, unauthorized-access automation | `docs/Roadmap.md` "Deferred / explicit non-goals" |

Jabari is one of the 14 QYVORA frameworks. It implements the shared CLI behaviour natively (it does **not** import `shared/qyvora-common`; see §10).

## 2. Audit scope and evidence sources

**Evidence actually consulted**

- Source tree: `cmd/`, `internal/` (26 packages), `pkg/` (`adb`, `android/adb`, `android/apk`, `models`, `utilities`).
- Documentation: `README.md` (7 910 B), all 14 `docs/*.md` (non-empty), `.github/workflows/{ci.yml,release.yml}`, `.goreleaser.yaml`, `Makefile`.
- Pre-captured help: `docs/audit/help/jabari.txt`.
- Contracts: `shared/qyvora-common/contract/contract.go`, `shared/qyvora-common/conformance/runner.go`, `knowledge/.../QYVORA-TOOL-OUTPUT-SPEC.md`.
- Historical audit: `knowledge/.../audits/QYVORA-FRAMEWORK-AUDIT.md` (findings re-tested, §13).
- Live binary probes (non-destructive, no device, no network) and `go test` / `go vet` — all recorded in §13.

**Not done (by constraint):** no device was attached or scanned; no USB/network/Bluetooth target was contacted; no live ADB session was opened. Device-path behaviour is therefore evidenced by source, unit tests and help output only.

## 3. Existing capability inventory

**CLI (tier grammar `<tool> <tier> <module> [<action>]`)** — verified with `--help`:

- `recon` — `assess` (`usb|ip|apk`), `discover`, `enumerate`, `analyze`, `target`, `tools`
- `exploit` — `poc`, `validate`
- `report` — render / `--list`
- flat — `capabilities`, `completion`, `script`, `updates`, `version`
- global flags: `-o/--output`, `--json`, `--events`, `--dry-run`, `--allow-shellout`, `-c/--config`, `--quiet`, `--verbose`

**Targets** (`internal/cli/target.go`): `usb`, `ip`, `apk`, `list`, `show`, with `-y/--authorized`.

**Transport layer** (`internal/transport/`)

| Transport | Mechanism | ADB binary required | Evidence |
|---|---|---|---|
| `USBTransport` | USB + local `adb` binary; surfaces `unauthorized` as `ErrUnauthorized` (RSA prompt) | yes | `usb.go:19-46` |
| `NetworkTransport` | raw TCP dial probe, then `adb connect` to `host:5555` (port overridable) | yes | `network.go:15-52`, `factory.go:62` |
| `NativeNetworkTransport` | native ADB protocol over TCP (no `adb` binary), opt-in `transport.native` | no | `network_native.go:18-56`, `cli/pipeline.go:29` |

**Assessment pipeline** (`internal/orchestration/pipeline.go:154,168`): `discovery → enumeration → analysis → validation → risk → reporting`, plus opt-in `poc` stage (`cli/pipeline.go:162-164`). APK targets use a separate static profile: `apk-analysis → analysis → validation → risk` (`cli/pipeline.go:157-159`, verified at runtime).

**Discovery** (`internal/discovery/stage.go` + transports): system-property fingerprinting — manufacturer, model, brand, product, Android version, API level, security patch, build fingerprint/ID, ABI, kernel (`uname -r`), `ro.adb.secure`, `ro.debuggable`, full property map, and root probe.

**Enumeration** (`internal/enumeration/stage.go:58,127-131`): `pm list packages` then per-package `dumpsys package <pkg>` with field parsers (`internal/enumeration/dumpsys.go`), run concurrently.

**APK engine** (`pkg/android/apk/apk.go`): ZIP container parse, binary AXML manifest decode, signing certificate extraction, DEX discovery, native libraries, `resources.arsc`, exported-component inference, dangerous-permission classification, ABIs, multi-dex, per-entry SHA-256 (`apk.go:209-243`). No `aapt`/`apktool` needed (native).

**Rules** (`internal/rules/builtin/builtin.go`): `AND-001` … `AND-010` and `AND-012` — **11 registered rules** (device: debuggable production build, outdated patch level, ADB unauthenticated, rooted device, test-keys build, outdated Android; app: backup enabled, cleartext traffic, debuggable app, excessive dangerous permissions, debug-signed APK).

**Validation** (`internal/validation/stage.go:36-98`): read-only `getprop` re-read of configuration evidence; concurrency-capped at 8; upgrades `detected → confirmed`.

**Risk** (`internal/risk/stage.go:27-36`): severity × confidence score written to `Session.RiskScore` / `RiskLevel` and persisted.

**Evidence** (`internal/evidence/store.go`): findings reference SHA-256-hashed artifacts, resolvable via `Resolve(hash)`.

**Reporting**: terminal, JSON, Markdown, HTML, YAML (`internal/reporting/reporting.go:27-60`), sessions persisted as `reports/session-<id>.json` (0600) (`cli/pipeline.go:103-113`).

**PoC modules** (`internal/poc/`): `debuggable`, `exported`, `worldreadable`; stage refuses to run without target authorization (`internal/poc/stage.go:40`).

**Toolchain detection** (`recon tools`): 15 tools probed — `adb`, `keytool`, `jarsigner` installed on this host; `aapt`, `aapt2`, `apksigner`, `zipalign`, `apktool`, `jadx`, `dex2jar`, `smali`, `frida`, `objection`, `drozer`, `sdkmanager` reported missing.

**Operational**: JSONL event stream (§2 envelope), config file + env (`QYVORA_` prefix, `internal/config/config.go:47-56`), `--dry-run`, self-update, `.qs` script engine with `--allow-shellout` gate, `updates`.

**Retired**: interactive console/REPL and per-tool TUI are `//go:build ignore` (`internal/cli/console.go:1`, `tui.go:1`, `console_ui.go:1`) and are not compiled into the binary.

## 4. Capability status matrix

Statuses: `IMPLEMENTED` / `PARTIALLY IMPLEMENTED` / `STUBBED` / `PLANNED` / `MISSING` / `UNVERIFIED` / `NOT APPLICABLE`.

| # | Capability | Status | Evidence |
|---|---|---|---|
| 1 | USB device detection + assessment (ADB binary) | IMPLEMENTED | `transport/usb.go`; `assess usb` help |
| 2 | Network (explicit IP) assessment, ADB port 5555 | IMPLEMENTED | `transport/network.go:15-52` |
| 3 | Native ADB protocol over TCP (no `adb` binary) | IMPLEMENTED | `network_native.go`; capabilities `status: native` |
| 4 | Device fingerprinting (getprop, kernel, ABI) | IMPLEMENTED | `network_native.go:80-108` |
| 5 | Rooted vs non-rooted detection | IMPLEMENTED (heuristic) | `network_native.go:150-157`, `usb.go:184-191` (`command -v su`) |
| 6 | Application/package enumeration (`pm`, `dumpsys`) | IMPLEMENTED | `enumeration/stage.go:58,127-141` |
| 7 | APK-only static analysis (manifest, cert, components, permissions, DEX) | IMPLEMENTED | `pkg/android/apk/apk.go`; E2E verified §13 |
| 8 | Config security-posture rules | IMPLEMENTED (11 rules) | `rules/builtin/builtin.go` |
| 9 | Evidence hashing + provenance | IMPLEMENTED | `evidence/store.go` |
| 10 | Validation of findings (read-only re-check) | IMPLEMENTED, **untested** | `validation/stage.go`; no `*_test.go` in package |
| 11 | Risk scoring persisted to session/report | IMPLEMENTED | `risk/stage.go:27-36`; runtime observed `52/100` |
| 12 | Reporting (terminal/JSON/MD/HTML/YAML) | IMPLEMENTED | all 5 formats executed, §13 |
| 13 | JSONL event stream | IMPLEMENTED (§2 conformant) | 17-event stream captured, §13 |
| 14 | Authorization gate (flag / env / TTY prompt / fail-closed) | IMPLEMENTED | `authorization.go:27-53`; exit 3 verified |
| 15 | PoC / exploitation-validation modules | PARTIALLY IMPLEMENTED (3 modules, device-path unexercised) | `internal/poc/*`; `exploit poc` help |
| 16 | Machine-readable capability manifest | PARTIALLY IMPLEMENTED (ability list, not the shared `CapabilityManifest` shape) | §10 |
| 17 | Wireless debugging (mDNS discovery, `adb pair`, pairing codes) | **MISSING** | no source reference (`rg mdns\|pair\|wireless` → none) |
| 18 | Network exposure beyond ADB (port/service enumeration) | **MISSING** (declared `docs/Roadmap.md` v0.3) | no scanner code; only one-port dial probe |
| 19 | Bluetooth discovery / profiles / permissions | **MISSING** | `rg -i bluetooth` → 0 matches in Go sources |
| 20 | Firmware / system-image analysis (boot.img, super.img, OTA) | **MISSING** | `rg boot\.img\|firmware\|system\.img` → 0 matches |
| 21 | Emulator-specific assessment | UNVERIFIED (transitively possible via `adb devices`) | only wording "emulated USB" `usb.go:14`; no emulator code |
| 22 | Fuzzer / intent-probing of exported components | PLANNED | `docs/Roadmap.md` v0.4 |
| 23 | CVE / known-vulnerability mapping | PLANNED | `docs/Roadmap.md` v0.4 |
| 24 | Runtime instrumentation (Frida/JADX/Apktool/Drozer) | STUBBED as *reported missing* (`status: missing`) | `capabilities` JSON; `recon tools` |
| 25 | Interactive console / per-tool TUI | NOT APPLICABLE (retired fleet-wide) | `//go:build ignore`; bare `jabari` → help, exit 0 |

## 5. Reconnaissance and discovery findings

- **Discovery is ADB-gated.** `discovery.Stage.Run` returns `transport.ErrNotConnected` when no transport is bound (`internal/discovery/stage.go:25-31`); every non-APK target must `Connect` first (`internal/cli/pipeline.go:37-41`). Nothing is collected by passive observation alone.
- **The only non-ADB byte on the wire is a TCP reachability probe.** Both network transports dial `host:5555` before speaking ADB (`network.go:47-52`, `network_native.go:38-49`). This yields *reachability of the ADB port only* — not service identification, not device identity.
- **No scanning.** There is no port range scan, no mDNS/Bonjour browse, no LAN sweep, no protocol fingerprinting. A target must be an explicitly supplied IP (`assess ip <addr>`, `target ip`).
- **Fingerprinting** is rich *once ADB is authorized*: full property map, build fingerprint, patch level, kernel, ABI, root heuristic (§3).
- **Enumeration** is a two-step `pm list packages` → `dumpsys package <pkg>` with parsers, run concurrently with a worker cap (`enumeration/stage.go:96`).

## 6. Analysis and assessment findings

- Rule evaluation is deterministic and unit-tested (`internal/rules/registry.go`, `builtin/builtin_test.go`); duplicate registration fails atomically (`internal/rules/registry_test.go`).
- Pipeline is profile-driven (`quick|standard|deep|application|device|network|compliance|research`) and aborts on first stage error (`internal/orchestration/pipeline.go`).
- APK runs execute a reduced profile with no device stages (`cli/pipeline.go:157-159`) — verified: stages `apk-analysis, analysis, validation, risk`.
- Redaction: **not applicable as a dedicated secret-sink**; findings carry evidence but Jabari has no secret-extraction module (domain is device/app posture, not credential harvesting).
- Reports are re-renderable from persisted sessions (`cli/report.go:31-125`), newest-first with `--list`.

## 7. Exploitation-related capability findings

- `exploit` tier exposes `poc` and `validate` only; there is no exploitation framework, no payload delivery, no persistence tooling (consistent with stated non-goals).
- PoC stage is **opt-in** (`--poc`) and **authorization-enforced inside the stage** (`internal/poc/stage.go:40` → exit 3), with `--poc-high-risk` further gating state-changing modules.
- Three modules exist: `debuggable`, `exported`, `worldreadable` (`internal/poc/`).
- **Validation-as-exploitation is foundation-stage only**: `internal/validation/stage.go` re-reads properties and upgrades confidence; "Part B" runtime/offensive validation is explicitly future work (`validation/stage.go:7-8`).
- **Not verified by execution:** PoC behaviour on a live target (no device in scope).

## 8. High-level implementation candidates

High-level = feasible with this machine, existing interfaces, ordinary dependencies (PROMPT §6).

| # | Candidate | Why it matters | Prerequisites | Priority |
|---|---|---|---|---|
| H1 | **Exposure probe beyond ADB**: connect-probe a caller-supplied *list* of TCP ports on an authorized IP target (HTTP/TLS/SSH/mDNS/banner grab) | PROMPT §5 "wireless and local-network exposure assessment"; currently a single hardcoded port | authorized IP target; `net.Dial` (already used) | **P1** |
| H2 | **Wireless debugging support**: `adb mdns services`, `adb pair <ip:port> <code>`, `adb connect` to the advertised non-5555 port | devices with wireless debugging enabled use ephemeral ports + pairing; current code only knows 5555 | `adb` binary ≥ 30 (or native protocol pairing); authorization | **P1** |
| H3 | **Bluetooth posture from the authorized device**: `dumpsys bluetooth_manager`, `settings get secure bluetooth_*`, enabled/profile state, nearby-bonded metadata reachable to shell | PROMPT §5 explicitly lists Bluetooth; requires no host radio — it is an on-device shell query | ADB-authorized device | **P1** |
| H4 | **Service enumeration via on-device shell** (`dumpsys -l`, `netstat`/`ss`, listening sockets) instead of host-side scanning | gives local-network exposure evidence without touching non-target hosts | ADB-authorized device | **P2** |
| H5 | **Emulator-aware target handling** (detect `ro.kernel.qemu`, `emulator-5554` serials, AVD profile) | unblocks repeatable CI assessment of Android targets | emulator image (SDK) | **P2** |
| H6 | **Expand APK static rules** (network-security-config, exported-provider path traversal, world-readable data dirs, `allowBackup` variants) + rule-ID `AND-011` reconciliation | static analysis needs no device at all — cheapest coverage gain | none | **P2** |
| H7 | **CVE/known-vuln mapping** against `ro.build.version.security_patch` / SDK level | turns fingerprinting into actionable risk | offline advisory dataset | **P3** |

## 9. Low-level deferred technique inventory

**DEFERRED — SPECIALISED PREREQUISITES.** None of the following are implemented; none are classified as irrelevant. Relevant to Jabari's Android domain.

| Technique | Objective | Hardware / access required | Feasibility here | Status | Why deferred / conditions to revisit |
|---|---|---|---|---|---|
| JTAG / SWD debug-interface access | Read/write SoC debug bus, extract memory | debug probe (J-Link/ST-Link), device teardown, schematics | Not feasible on this host | **DEFERRED — SPECIALISED PREREQUISITES** | needs lab hardware + disposable units |
| UART / serial console analysis | Bootloader & kernel console, unsigned shell | logic level adapter, UART pads, board knowledge | Not feasible | **DEFERRED — SPECIALISED PREREQUISITES** | requires physical teardown and per-device pinouts |
| Chip-off / hardware-assisted flash extraction | Raw eMMC/UFS image recovery | cleanroom-capable BGA rework, chip readers | Not feasible | **DEFERRED — SPECIALISED PREREQUISITES** | destructive, device-destroying, specialist facility |
| RF / SDR / baseband analysis | Cellular, Wi-Fi, BLE at the radio layer | SDR hardware (HackRF/USRP), RF shielding, expertise | Not feasible | **DEFERRED — SPECIALISED PREREQUISITES** | needs RF hardware; also regulatory constraints |
| Baseband/firmware research in dedicated environments | Modem firmware vulnerability research | vendor toolchains, emulator rigs, disclosure process | Not feasible | **DEFERRED — SPECIALISED PREREQUISITES** | exceptional privileges + long research cycles |
| Fault injection / side-channel analysis | Glitching, power/EM analysis of secure elements | lab instrumentation, decapping | Not feasible | **DEFERRED — SPECIALISED PREREQUISITES** | hardware lab only |
| Bootloader/unlock & custom-recovery deep inspection (signed images, verified boot chain states beyond `getprop`) | Verified-boot chain validation | unlocked units, OEM-specific image sources | Partially reachable via props; full chain not | **DEFERRED** (image sources are device-specific) | revisit if firmware-image corpus becomes available |

**Investigated and confirmed absent from the codebase** (no special hardware needed, but currently unimplemented — these belong to §8 high-level, listed here because PROMPT §5 asks explicitly): Bluetooth (0 source matches), firmware/system-image parsing (0 matches), mDNS/wireless-debugging discovery (0 matches), emulator-specific logic (0 matches).

### Technique-to-mechanism mapping (PROMPT §5)

| # | Technique | Mechanism | Without ADB? | Requires privileged/authorized interface? | Status |
|---|---|---|---|---|---|
| T1 | USB device detection & enumeration | **USB** (via `adb` binary) | No | Yes — USB attach + RSA authorization (`usb.go:36-45`) | IMPLEMENTED |
| T2 | Network IP-based assessment | **network** — TCP dial + ADB | Port-reachability only without ADB | Yes — device must have `adb tcpip` enabled and host authorized | IMPLEMENTED |
| T3 | Native ADB protocol session | **network** (protocol implemented in-process) | No (still an authorized ADB session) | Yes | IMPLEMENTED |
| T4 | Wireless debugging / pairing | **network + privileged pairing** | n/a | Yes — pairing code from the device screen | **MISSING** (H2) |
| T5 | Bluetooth discovery/profiles | **Bluetooth** (host radio) *or* on-device shell query | No host-radio path exists | On-device shell ⇒ ADB | **MISSING** (H3) |
| T6 | APK/manifest/signing analysis | **APK-only** | **Yes — fully offline** | No | IMPLEMENTED |
| T7 | OS fingerprinting (build, patch, ABI, kernel) | **privileged interface** (`getprop`) | No | Yes — ADB | IMPLEMENTED |
| T8 | App/service enumeration (`pm`, `dumpsys`) | **privileged interface** (shell) | No | Yes — ADB | IMPLEMENTED |
| T9 | Wireless/local-network exposure | **network** (host-side) / on-device `ss` | Partial: one ADB-port probe only | Host probe needs no ADB; anything more needs ADB or scanning scope | PARTIALLY IMPLEMENTED (H1/H4) |
| T10 | Device security posture (root, ADB secure, debuggable, SELinux) | **privileged interface** | No | Yes — ADB | IMPLEMENTED (SELinux not yet read; Roadmap v0.3) |
| T11 | Rooted vs non-rooted limits | **privileged interface** (`command -v su`) | No | Yes — ADB | IMPLEMENTED (heuristic; no Magisk-hiding detection) |
| T12 | Firmware / system-image analysis | **image file analysis** (offline, if images supplied) | Yes, given an image artifact | No (once an image is in hand) | **MISSING** (no parser) |
| T13 | Emulator assessment | **USB/TCP as presented by `adb`** | No | Same as T1/T2 | UNVERIFIED |
| T14 | Developer-mode / authorization requirements | **operator workflow** | n/a | Yes — `--authorized`/env/TTY, fail-closed exit 3 | IMPLEMENTED (`authorization.go:27-53`) |
| T15 | JTAG/SWD/UART/chip-off/RF/SDR/baseband | **hardware interface** | No | Exceptional physical access | **DEFERRED — SPECIALISED PREREQUISITES** (§9) |

**Boundary statement (PROMPT §5, explicit).** Jabari can reach *no* private device content from a bare IP address. Knowing a device's IP yields exactly one bit — whether TCP `host:5555` accepts a connection (`network.go:47-52`). Everything else in T2/T7/T8/T10/T11 is obtained only after an authorized ADB session, which requires the device to have enabled debugging over TCP **and** the host to have been accepted (RSA prompt / pairing). Offline APK analysis (T6) is the only technique that requires neither a device nor any privilege.

## 10. Common-layer conformance

Frameworks implement the shared contract natively — **no `shared/qyvora-common` import exists in this repository** (verified by import search). Conformance is therefore behavioural.

| Dimension | Status | Evidence |
|---|---|---|
| Identity/version metadata (`framework, version, commit, date, build_user, go_version, arch, os, website, support, built_in`) | **SHARED** | `version -o json` matches `QYVORA-TOOL-OUTPUT-SPEC.md` §4 exactly |
| Exit codes 0/1/2/130 | **SHARED** | bad flag → 2, bad command → 2, `--help` → 0 (executed) |
| Exit code 3 used for "refused / unsupported" | **SHARED (intent)** | `authorization.go:48,51`, `poc/stage.go:40` → exit 3 verified |
| JSONL event envelope §2 | **SHARED** | `schema_version/timestamp/execution_id/framework/level/event/data` captured |
| Severity / confidence / finding-status vocabularies | **SHARED** | `pkg/models`, report output |
| Report JSON shape | **FRAMEWORK-SPECIFIC** (documented as such in spec §6 `jabari (pkg/models)`) | snake_case session document |
| Capability manifest shape | **INCONSISTENT** | `jabari capabilities -o json` → bare array `{name, category, status, description}`; shared `contract.CapabilityManifest` expects `{framework, version, commands[{name, inputs, outputs, dangerous, authorization, emits, deterministic}]}`. Jabari emits no command-level metadata and no exit-code table |
| Exit-code declaration in capability document | **MISSING** | capabilities JSON carries no `exit_codes` block (Imhotep does) |
| `--events` stdout + machine format collision guard | **FRAMEWORK-SPECIFIC but well-behaved** | `root.go:97-104` → exit 2 |
| Script engine (`.qs`) + `--allow-shellout` gate | **SHARED / DUPLICATED** | `script.go`, same gate name as Imhotep |
| Self-update | **DUPLICATED** (per-framework `internal/selfupdate`) | both repos carry their own |
| Conformance runner checks | **SHARED** | ran against this binary — §13 |
| Retired TUI/session linkage | **SHARED** | source `//go:build ignore`; binary check UNVERIFIED (stripped) |

**Conformance verdict: PARTIALLY CONFORMANT.** Behavioural contract (identity, exit codes, events, formats) conforms and was executed; the machine capability manifest does not match the shared `CapabilityManifest` schema and omits exit-code metadata.

## 11. AI orchestration readiness

| Requirement | Status | Evidence |
|---|---|---|
| Stable framework identity | CONFORMANT | `version -o json` |
| Version/compatibility metadata | CONFORMANT | full build block incl. `go_version` |
| Machine-readable capability discovery | PARTIALLY CONFORMANT | `capabilities -o json` works but is an ability list, not operations with inputs/outputs/risk (§10) |
| Explicit operation inputs/outputs | PARTIALLY CONFORMANT | `--help` examples are good; not machine-readable |
| Predictable exit codes/errors | CONFORMANT | 0/1/2/3/130 verified; error text on stderr |
| Structured results & evidence | CONFORMANT | session JSON + hashed evidence (`reports/session-*.json`) |
| Prerequisite declarations | PARTIALLY CONFORMANT | `recon tools` reports 15 tools installed/missing (machine-readable) |
| Permission/authorization requirements | CONFORMANT | fail-closed exit 3, `--authorized`/`QYVORA_AUTHORIZED` |
| Timeouts & cancellation | CONFORMANT | `timeout` flag; context bound to SIGINT/SIGTERM (`root.go` `ExecuteArgs`) |
| Safe defaults | CONFORMANT | `--dry-run`; PoC opt-in; authorization gate |
| Read-only vs state-changing distinction | PARTIALLY CONFORMANT | `Command.Dangerous` field exists in the shared contract but is not emitted |
| Risk classification of operations | NOT APPLICABLE (no safety-op registry; Imhotep has one) | — |
| Log/result correlation | CONFORMANT | `execution_id` = session id across events and result |
| Honest unsupported reporting | CONFORMANT | `capabilities` `status: missing` for Frida/JADX/Apktool/Drozer; `recon tools` per-tool status |
| Conformance-runner green | PARTIALLY CONFORMANT | 5/6 checks pass; `tui-retired` UNVERIFIED (stripped binary) |

**Recommended future architecture (proposal only, not implemented):** emit `contract.CapabilityManifest` from `jabari capabilities -o json` (framework/version/commands with inputs, outputs, `dangerous`, `authorization`, `emits`, `deterministic`), add an `exit_codes` block, and expose per-operation safety metadata so an orchestrator can decide between `assess`, `poc`, and read-only `discover`/`enumerate` without parsing help text.

## 12. CLI, repository-layout, and documentation findings

**CLI** — well-formed and verified: tier grammar present; unknown command/flag → 2; `--help` → 0; bare invocation prints help and exits 0 (REPL retired); invalid `-f xml` → 2; `--events off` accepted; `--events stdout` + machine format → 2.

**Layout** — modular (`cmd/`, `internal/`, `pkg/`), `bin/` and `continue_AI.sh` gitignored (`.gitignore:2,6`), `scripts/verify-artifact.sh` (8 512 B) present, `assets/`, `configs/`, `.goreleaser.yaml`, Makefile, CI (`ci.yml`, `release.yml`). **No zero-byte files anywhere in the repository** (`find -size 0` → empty). The historical empty `plugins/` and `test/` placeholders no longer exist.

**Documentation problems (misleading/outdated — P1)**

1. `README.md:116-137` — Quick start instructs "Drop into the interactive console (Metasploit-style REPL): `jabari`" and shows a `jabariλ >` prompt with tab completion. Actual: bare `jabari` prints help/banner and exits 0; `jabari console` → **exit 2, unknown command**. Console sources are `//go:build ignore`.
2. `README.md:40-45` and `docs/Roadmap.md` ("Next (v0.2: Static analysis)") both describe APK analysis, manifest parsing and signing inspection as *future work* — all are implemented (`jabari assess apk`, `pkg/android/apk`, AND-007/AND-012).
3. `README.md:54` claims "initial set: `AND-001` … `AND-007`" — 11 rules are registered (`AND-001`–`AND-010`, `AND-012`; **`AND-011` is skipped**, an unexplained ID gap).
4. `README.md:143-160` example terminal output is branded `ANDROIDSEC`, inconsistent with the current `jabari` banner.
5. `docs/Roadmap.md` `docs/` set is otherwise complete and non-empty (14 docs).

**Repository hygiene** — clean; only the three pre-existing staged renames into `docs/IMPLEMENTATION-HISTORY/` are present in the index (left untouched).

## 13. Testing and verification findings

**Commands actually executed (all non-destructive; no device, no network target):**

| Check | Result |
|---|---|
| `go vet ./...` | **exit 0**, no findings |
| `go test ./... -count=1 -timeout 180s` | **PASS** — **22 packages `ok`, 10 with no test files, 0 failures** (measured twice) |
| `go test ./internal/rules/ -run TestRegistryErrorTolerated -count=30` | **PASS ×30** (historical flake fixed — assertion now locates the diagnostic by `RuleID`, `registry_test.go:85-94`) |
| `bin/jabari version -o json` | exit 0, full §4 metadata |
| `bin/jabari capabilities -o json` | exit 0, 21 capability records (3 192 B) |
| `bin/jabari --help` | exit 0 |
| `bin/jabari --badflag` / `bogus-command` | **exit 2 / exit 2** |
| `bin/jabari assess apk <fixture>` without `-y`, non-TTY stdin | **exit 3**, `authorization declined; assessment aborted` |
| `bin/jabari assess apk examples/fixtures/debuggable.apk -y -o json --events <file>` | **exit 0**; 2 findings; stages `apk-analysis, analysis, validation, risk`; risk `52/100 (medium)`; 17 JSONL events; session written to `QYVORA_REPORT_DIR` (redirected to `/tmp`) |
| `bin/jabari report -f {json,markdown,yaml,html,terminal}` | **exit 0 each** (2 618 / 1 478 / 2 486 / 2 379 / 508 B) |
| `bin/jabari report -f xml` | **exit 2** (invalid format is a usage error) |
| `bin/jabari report --list` | exit 0, lists session |
| `bin/jabari recon tools -o json` | exit 0, 15 tools (3 installed / 12 missing) |
| `bin/jabari recon` / `exploit` / `target --help` / `assess apk --help` | exit 0, tier structure confirmed |
| Shared conformance runner (`qyvora-conformance -bin …`) | **5/6 checks PASS**: `version-semver`, `version-exit0`, `usage-exit-2`, `unknown-cmd-exit-2`, `capabilities`; **`tui-retired` = UNVERIFIED** — reason: *stripped ELF* ("no symbol section"), not a detected violation. Source confirms `//go:build ignore` on `tui.go`, `console.go`, `console_ui.go` |

**Historical findings (2026-08) re-verified:**

| Historical finding | Verdict | Evidence |
|---|---|---|
| Flaky `TestRegistryErrorTolerated` (map-order) | **FIXED** | `registry_test.go:85-94`; 30/30 runs pass |
| CI pinned Go 1.21/1.22 vs `go 1.26.5` | **FIXED** | `.github/workflows/ci.yml:29`, `release.yml:56` → `go-version-file: go.mod` |
| Module path `anomalyco/qyvora-jabari` | **FIXED** | `go.mod:1` = `github.com/QYVORA/qyvora-jabari` |
| Risk score not persisted | **FIXED** | `risk/stage.go:29-33` writes `env.Session.RiskScore`/`RiskLevel`; observed in output |
| `-o json` ignored | **FIXED** | `root.go:308-313` + `report.go:95-109` `resolveReportFormat()` on every render path; verified by execution |
| Untracked 12 MB binary + `continue_AI.sh` | **FIXED** | `.gitignore:2` (`bin/`), `:6` (`continue_AI.sh`) — `git status --ignored` shows both ignored |
| Empty `plugins/`, `reports/`, `test/` placeholders | **RESOLVED (2) / N/A (1)** | `plugins/`, `test/` no longer exist; `reports/` is a runtime output directory created on demand (not tracked) |

**Coverage gaps**

- `internal/validation` has **no test files** — the confirmed-finding upgrade path is untested.
- The other 9 packages without tests: `cmd/jabari`, `internal/analysis`, `internal/capabilities`, `internal/core`, `internal/discovery`, `internal/errors`, `internal/evidence`, `internal/i18n`, `internal/logger` (10 total incl. `validation`).
- No integration/E2E test exercises the device pipeline (`discovery → enumeration`) — only parser-level tests (`enumeration/stage_test.go`, `dumpsys_test.go`) and transport unit tests (`transport_test.go`, all with fakes).
- Emulator path (T13) never exercised.

**Effect on the working tree:** `git status` before and after the audit shows only the three pre-existing staged renames; `bin/` and `continue_AI.sh` remain ignored. An empty `reports/` directory inadvertently created by one probe was detected and removed; no tracked file, dependency, config or lockfile changed. No writes to `~/.config/qyvora/`.

## 14. Prioritized gaps and recommendations

**P0 — Critical:** none found. No blocking defect, no unsafe default, no data-loss path was identified.

| ID | Priority | Gap | Evidence | Recommended action (future phase) |
|---|---|---|---|---|
| G1 | **P1** | Non-ADB/extra-ADB technique coverage is absent: Bluetooth, wireless-debugging pairing, exposure beyond the ADB port, firmware-image analysis | §4 rows 17-20; 0 source matches | Implement H1-H3 first (all high-level), then H4 |
| G2 | **P1** | README/Roadmap materially mislead: document a retired REPL as the primary workflow and describe shipped APK analysis as "next" | §12 items 1-4 | Rewrite Quick start; re-baseline Roadmap; reconcile `AND-011` |
| G3 | **P2** | Capability manifest does not implement `contract.CapabilityManifest` and omits exit codes | §10 | Emit the shared shape (orchestration blocker) |
| G4 | **P2** | `internal/validation` untested; device pipeline has no integration test | §13 | Add unit tests for confirmation/refutation; fixture-driven pipeline test |
| G5 | **P2** | Emulator assessment unverified; no emulator detection | §4 row 21, H5 | Add `ro.kernel.qemu` detection + emulator target docs |
| G6 | **P3** | Rule-ID gap at `AND-011`; capabilities `status` vocabulary (`native/integration/missing`) differs from the fleet's status vocabulary | `builtin.go`, capabilities JSON | Document the vocabulary or align with §4 statuses |
| G7 | **P3** | `recon tools` reports 12 missing optional binaries with no machine-readable install guidance | `tools` output | Emit package/install hints as JSON fields |

## 15. Dependencies, prerequisites, and unresolved questions

**Runtime dependencies** (`go.mod`): `spf13/cobra`, `spf13/viper`, plus internal packages. Optional external binaries: `adb` (required for USB and for the default network transport), `keytool`/`jarsigner` (reported), and the "missing" set (aapt, aapt2, apksigner, zipalign, apktool, jadx, dex2jar, smali, frida, objection, drozer, sdkmanager).

**Prerequisites for device techniques:** physical/USB attach or an explicitly authorized IP; device debugging enabled (`adb tcpip` for T2/T4); host authorization accepted (RSA/pairing); `--authorized` or `QYVORA_AUTHORIZED=true` for non-interactive runs.

**Unresolved questions**

1. Native ADB transport is opt-in (`transport.native`) and TCP-only — is a native USB transport intended to replace the `adb` binary dependency (capabilities currently reports USB as `integration`)?
2. What is the intended replacement for the skipped `AND-011` rule ID?
3. Should exit 3 remain "authorization declined", or should authorization failures map to a distinct code? (Currently consistent with the spec's "unsupported / refused" wording, but the spec's jabari enforcement note lists only 2 and 130.)
4. Emulator support: intended as a first-class target type or as "whatever `adb devices` reports"?
5. Stripped release binaries prevent the conformance runner's symbol-based `tui-retired` assertion — should the runner fall back to source/`go version -m` evidence?

## 16. Statement of compliance

**No feature implementation was performed.** This document is the product of read-only source inspection, help/capability introspection, non-destructive local execution, and existing test/vet runs. No application code, test, configuration, dependency, lockfile, workflow or existing documentation was modified; no device, network, Bluetooth or other target was contacted; no specialist technique was implemented; and the only file created by this audit is this `AUDIT.md`. The three pre-existing staged renames into `docs/IMPLEMENTATION-HISTORY/` remain staged and untouched.
