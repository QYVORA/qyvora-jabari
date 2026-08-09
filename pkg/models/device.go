package models

// DeviceInfo is the normalized device metadata produced by the discovery
// stage. All fields are strings (even Android version and API level) because
// the source data arrives as strings from ADB and can be sparse; consumers
// parse them as needed.
type DeviceInfo struct {
	Manufacturer     string            `json:"manufacturer,omitempty"`
	Model            string            `json:"model,omitempty"`
	Brand            string            `json:"brand,omitempty"`
	Product          string            `json:"product,omitempty"`
	AndroidVersion   string            `json:"android_version,omitempty"`
	APILevel         string            `json:"api_level,omitempty"`
	SecurityPatch    string            `json:"security_patch,omitempty"`
	BuildFingerprint string            `json:"build_fingerprint,omitempty"`
	BuildID          string            `json:"build_id,omitempty"`
	Kernel           string            `json:"kernel,omitempty"`
	Architecture     string            `json:"architecture,omitempty"`
	Serial           string            `json:"serial,omitempty"`
	ABI              string            `json:"abi,omitempty"`
	Rooted           bool              `json:"rooted,omitempty"`
	DebugState       string            `json:"debug_state,omitempty"`
	RoAdbSecure      string            `json:"ro_adb_secure,omitempty"`
	SystemProperties map[string]string `json:"system_properties,omitempty"`
}

// Summary returns a compact single-line description used in logs and the
// terminal summary.
func (d *DeviceInfo) Summary() string {
	if d == nil {
		return "no device information"
	}
	parts := make([]string, 0, 4)
	if d.Manufacturer != "" {
		parts = append(parts, d.Manufacturer)
	}
	if d.Model != "" {
		parts = append(parts, d.Model)
	}
	if d.AndroidVersion != "" {
		parts = append(parts, "Android "+d.AndroidVersion)
	}
	if d.APILevel != "" {
		parts = append(parts, "API "+d.APILevel)
	}
	s := ""
	for i, p := range parts {
		if i > 0 {
			s += " "
		}
		s += p
	}
	if s == "" {
		return "unknown Android device"
	}
	return s
}
