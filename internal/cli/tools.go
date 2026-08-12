package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// toolSpec describes one Android security toolchain component.
type toolSpec struct {
	name    string
	purpose string
	bins    []string // binary names to look for on PATH
	kaliPkg string   // the Kali Linux package that provides it
}

// toolsSpecs is the recognized Android assessment toolchain. Only tools the
// operator explicitly invokes are listed; detection is passive (PATH lookup)
// and never runs the tools.
var toolsSpecs = []toolSpec{
	{name: "adb", purpose: "Android Debug Bridge (device access)", bins: []string{"adb"}, kaliPkg: "android-tools-adb"},
	{name: "aapt", purpose: "APK manifest/asset inspection", bins: []string{"aapt"}, kaliPkg: "aapt"},
	{name: "aapt2", purpose: "APK manifest/asset inspection (v2)", bins: []string{"aapt2"}, kaliPkg: "aapt2"},
	{name: "apksigner", purpose: "APK signing certificate validation", bins: []string{"apksigner"}, kaliPkg: "apksigner"},
	{name: "zipalign", purpose: "APK alignment verification", bins: []string{"zipalign"}, kaliPkg: "zipalign"},
	{name: "apktool", purpose: "APK decoding and repackaging", bins: []string{"apktool"}, kaliPkg: "apktool"},
	{name: "jadx", purpose: "APK/DEX decompiler", bins: []string{"jadx", "jadx-cli"}, kaliPkg: "jadx"},
	{name: "dex2jar", purpose: "DEX to JAR conversion", bins: []string{"d2j-dex2jar", "dex2jar"}, kaliPkg: "dex2jar"},
	{name: "smali", purpose: "DEX assembler/disassembler", bins: []string{"smali", "baksmali"}, kaliPkg: "smali"},
	{name: "frida", purpose: "Runtime instrumentation/hooking", bins: []string{"frida", "frida-ps"}, kaliPkg: "frida-tools"},
	{name: "objection", purpose: "Frida-based runtime exploration", bins: []string{"objection"}, kaliPkg: "objection"},
	{name: "drozer", purpose: "Android attack surface testing", bins: []string{"drozer"}, kaliPkg: "drozer"},
	{name: "keytool", purpose: "Keystore/signing key inspection", bins: []string{"keytool"}, kaliPkg: "default-jdk"},
	{name: "jarsigner", purpose: "JAR/APK signature inspection", bins: []string{"jarsigner"}, kaliPkg: "default-jdk"},
	{name: "sdkmanager", purpose: "Android SDK component manager", bins: []string{"sdkmanager"}, kaliPkg: "google-android-platform-tools-installer"},
}

// toolStatus is one detection result.
type toolStatus struct {
	Name      string `json:"name"`
	Purpose   string `json:"purpose"`
	Path      string `json:"path,omitempty"`
	KaliPkg   string `json:"kali_package,omitempty"`
	Installed bool   `json:"installed"`
}

// detectTools looks up each toolchain component on PATH. Detection is purely
// passive: it never executes the tools, only resolves their location.
func detectTools() []toolStatus {
	status := make([]toolStatus, 0, len(toolsSpecs))
	for _, spec := range toolsSpecs {
		st := toolStatus{Name: spec.name, Purpose: spec.purpose, KaliPkg: spec.kaliPkg}
		for _, bin := range spec.bins {
			if path, err := exec.LookPath(bin); err == nil {
				st.Path = path
				st.Installed = true
				break
			}
		}
		status = append(status, st)
	}
	return status
}

// newToolsCmd implements the one-shot "jabari tools" command: it reports the
// Android assessment toolchain and whether each component is available.
func newToolsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tools",
		Short: "Report the Android assessment toolchain",
		Long: `Report which Android security tools are installed and available on this
host. Detection is passive PATH lookup; nothing is executed.

Typical on a Kali host:
  apt install android-tools-adb apktool jadx frida-tools objection drozer`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rows := make([][]string, 0, len(toolsSpecs))
			for _, st := range detectTools() {
				status := "missing"
				location := ""
				if st.Installed {
					status = "installed"
					location = st.Path
				}
				rows = append(rows, []string{st.Name, status, st.Purpose, location, st.KaliPkg})
			}
			printer.PrintTable([]string{"tool", "status", "purpose", "path", "kali package"}, rows)
			return nil
		},
	}
}

// cmdTools prints the toolchain report in the interactive console.
func (c *jabariConsole) cmdTools() {
	rows := make([][]string, 0, len(toolsSpecs))
	for _, st := range detectTools() {
		status := c.ui.Amber("missing")
		location := "-"
		if st.Installed {
			status = c.ui.Green("installed")
			location = st.Path
		}
		rows = append(rows, []string{st.Name, status, st.Purpose, location, st.KaliPkg})
	}
	c.ui.Section("Android assessment toolchain")
	c.ui.Table([]string{"tool", "status", "purpose", "path", "kali package"}, rows)
	c.ui.Status(">", "install missing tools with: apt install android-tools-adb apktool jadx frida-tools objection drozer")
}

// toolsSummary is a compact one-line hint for the HUD and prompts.
func toolsSummary() string {
	var missing []string
	for _, st := range detectTools() {
		if !st.Installed {
			missing = append(missing, st.Name)
		}
	}
	if len(missing) == 0 {
		return "toolchain complete"
	}
	return fmt.Sprintf("missing: %s", strings.Join(missing, ", "))
}
