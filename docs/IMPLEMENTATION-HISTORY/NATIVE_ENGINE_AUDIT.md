# JABARI Native Engine Audit

## Current State

### Architecture
- **Pipeline**: discovery → enumeration → analysis → validation → risk → reporting
- **Transport abstraction**: USB/Network both shell out to `adb` binary
- **Models**: Solid foundation (Device, Application, Finding, Evidence, Session)
- **Rules**: 7 builtin rules (AND-001 to AND-007)
- **Tests**: All passing ✓

### External Dependencies
**Current hard dependencies:**
- `adb` - ALL device communication (USB/TCP transport)

**Listed but not yet used in core:**
- aapt, aapt2, apksigner, zipalign (APK analysis - roadmap)
- apktool, jadx, dex2jar, smali (decompilation - roadmap)
- frida, objection, drozer (runtime - roadmap)

### Code Paths Using exec.Command
1. `pkg/adb/adb.go` - All ADB operations shell out
2. `internal/transport/usb.go` - Uses adb.Client
3. `internal/transport/network.go` - Uses adb.Client
4. `internal/cli/shell.go` - Interactive shell (OK, not core)

## Migration Plan

### Phase 1: Native ADB Protocol (PRIORITY 1)
**Target**: `pkg/android/adb/`
- Protocol: CNXN, AUTH, OPEN, OKAY, WRTE, CLSE
- USB transport via libusb or gousb
- TCP transport (direct socket)
- Replace `pkg/adb` shell-out gradually

### Phase 2: APK Parser (PRIORITY 2)
**Target**: `pkg/android/apk/`
- ZIP parsing (stdlib archive/zip)
- Binary XML parser (AndroidManifest.xml)
- Certificate extraction (stdlib crypto)
- Extend `models.Application`

### Phase 3: DEX Parser (PRIORITY 5)
**Target**: `pkg/android/dex/`
- Structural analysis only (not decompilation)
- API call detection
- String extraction

### Phase 4: Capability System
**Replace**: `internal/cli/tools.go`
- Native capabilities vs optional integrations
- Runtime capability detection

## Package Structure
```
pkg/android/
  adb/          # Native ADB protocol
    protocol.go
    connection.go
    auth.go
    usb.go
    tcp.go
  apk/          # APK parsing
    apk.go
    manifest.go
    binary_xml.go
    resources.go
  dex/          # DEX analysis
    dex.go
    parser.go
  signing/      # Certificate/signature
    signing.go

internal/integrations/  # Optional external tools
  frida/
  jadx/
```

## Risk Areas
- USB transport: Needs libusb or direct device access
- Binary XML: Complex format, needs careful parsing
- ADB auth: RSA key handling
- DEX format: Large specification
