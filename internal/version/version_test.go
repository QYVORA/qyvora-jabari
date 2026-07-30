package version

import "testing"

func TestString(t *testing.T) {
	if String() != Version {
		t.Errorf("String() = %q, want %q", String(), Version)
	}
}

func TestGetInfo(t *testing.T) {
	info := GetInfo()
	if info.Version != Version {
		t.Errorf("GetInfo().Version = %q, want %q", info.Version, Version)
	}
	if info.Commit != Commit {
		t.Errorf("GetInfo().Commit = %q, want %q", info.Commit, Commit)
	}
}
