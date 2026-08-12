package cli

import (
	"testing"
)

func TestDetectTools(t *testing.T) {
	status := detectTools()
	if len(status) == 0 {
		t.Fatal("detectTools returned nothing")
	}
	want := map[string]bool{"adb": true, "apktool": true, "jadx": true, "frida": true, "objection": true, "drozer": true}
	seen := map[string]bool{}
	for _, st := range status {
		seen[st.Name] = true
		if st.Name == "" {
			t.Error("tool with empty name")
		}
		if st.Purpose == "" {
			t.Errorf("%s: missing purpose", st.Name)
		}
		if st.KaliPkg == "" {
			t.Errorf("%s: missing kali package", st.Name)
		}
		if st.Installed && st.Path == "" {
			t.Errorf("%s: installed but no path", st.Name)
		}
		if !st.Installed && st.Path != "" {
			t.Errorf("%s: not installed but has path %q", st.Name, st.Path)
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("toolchain missing %q", name)
		}
	}
}
