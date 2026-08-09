package models

// Application is the normalized inventory entry for an installed app or an
// analyzed APK. Fields are sparse by design; collectors fill in whatever they
// observed and leave the rest empty.
type Application struct {
	PackageName      string            `json:"package_name"`
	VersionName      string            `json:"version_name,omitempty"`
	VersionCode      string            `json:"version_code,omitempty"`
	Source           string            `json:"source,omitempty"`
	SystemApp        bool              `json:"system_app"`
	Debuggable       bool              `json:"debuggable"`
	AllowBackup      bool              `json:"allow_backup"`
	Permissions      []string          `json:"permissions,omitempty"`
	Activities       []string          `json:"activities,omitempty"`
	Services         []string          `json:"services,omitempty"`
	Receivers        []string          `json:"receivers,omitempty"`
	Providers        []string          `json:"providers,omitempty"`
	HasWebView       bool              `json:"has_webview"`
	CleartextTraffic bool              `json:"cleartext_traffic"`
	SignerSHA256     []string          `json:"signer_sha256,omitempty"`
	UsesCleartext    bool              `json:"uses_cleartext"`
	Attributes       map[string]string `json:"attributes,omitempty"`
}

// RiskPermissions is a coarse classification of Android permissions used by
// the permission analysis rules. It is intentionally conservative.
var RiskPermissions = map[string]string{
	"android.permission.CAMERA":                   "dangerous",
	"android.permission.READ_CONTACTS":            "dangerous",
	"android.permission.WRITE_CONTACTS":           "dangerous",
	"android.permission.ACCESS_FINE_LOCATION":     "dangerous",
	"android.permission.ACCESS_COARSE_LOCATION":   "dangerous",
	"android.permission.RECORD_AUDIO":             "dangerous",
	"android.permission.READ_SMS":                 "dangerous",
	"android.permission.RECEIVE_SMS":              "dangerous",
	"android.permission.SEND_SMS":                 "dangerous",
	"android.permission.READ_EXTERNAL_STORAGE":    "dangerous",
	"android.permission.WRITE_EXTERNAL_STORAGE":   "dangerous",
	"android.permission.BODY_SENSORS":             "dangerous",
	"android.permission.READ_PHONE_STATE":         "dangerous",
	"android.permission.CALL_PHONE":               "dangerous",
	"android.permission.SYSTEM_ALERT_WINDOW":      "special",
	"android.permission.WRITE_SETTINGS":           "special",
	"android.permission.REQUEST_INSTALL_PACKAGES": "special",
	"android.permission.MANAGE_EXTERNAL_STORAGE":  "special",
	"android.permission.INTERNET":                 "normal",
}

// PermissionRisk returns a coarse classification for a permission, defaulting
// to "unknown" when the permission is not in the local table.
func PermissionRisk(perm string) string {
	if r, ok := RiskPermissions[perm]; ok {
		return r
	}
	return "unknown"
}
