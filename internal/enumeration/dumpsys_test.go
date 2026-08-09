package enumeration

import "testing"

func TestParsePackageDump(t *testing.T) {
	dump := `Activity Resolver Table:
  Full MIME types:
Packages:
  Package [com.example.app] (a1b2):
    userId=10042
    pkg=Package{...}
    codePath=/data/app/com.example.app/base.apk
    versionCode=25 minSdk=23 targetSdk=35
    versionName=2.4.1
    pkgFlags=[ DEBUGGABLE ]
    dataDir=/data/user/0/com.example.app
    requested permissions:
      android.permission.INTERNET
      android.permission.CAMERA
      android.permission.ACCESS_FINE_LOCATION
    install permissions:
      android.permission.INTERNET
`
	app := parsePackageDump(dump, "com.example.app")

	if app.VersionName != "2.4.1" {
		t.Errorf("VersionName = %q, want 2.4.1", app.VersionName)
	}
	if app.VersionCode != "25" {
		t.Errorf("VersionCode = %q, want 25", app.VersionCode)
	}
	if !app.Debuggable {
		t.Error("Debuggable = false, want true (pkgFlags contains DEBUGGABLE)")
	}
	if app.SystemApp {
		t.Error("SystemApp = true, want false")
	}
	if len(app.Permissions) != 3 {
		t.Fatalf("Permissions = %v, want 3 entries", app.Permissions)
	}
	if app.Permissions[0] != "android.permission.INTERNET" {
		t.Errorf("Permissions[0] = %q", app.Permissions[0])
	}
	if app.Attributes["uid"] != "10042" {
		t.Errorf("uid attribute = %q, want 10042", app.Attributes["uid"])
	}
}

func TestParsePackageDumpSystemApp(t *testing.T) {
	dump := `Packages:
  Package [com.android.settings] (x1):
    userId=1000
    versionCode=35
    pkgFlags=[ SYSTEM ALLOW_CLEAR_USER_DATA ]
`
	app := parsePackageDump(dump, "com.android.settings")
	if !app.SystemApp {
		t.Error("SystemApp = false, want true")
	}
	if app.Debuggable {
		t.Error("Debuggable = true, want false")
	}
}

func TestParsePackageList(t *testing.T) {
	out := "package:com.a\npackage:com.b\n\nwarning: garbage line\n"
	pkgs := parsePackageList(out)
	if len(pkgs) != 2 || pkgs[0] != "com.a" || pkgs[1] != "com.b" {
		t.Errorf("parsePackageList = %v, want [com.a com.b]", pkgs)
	}
}
