package models

import "time"

// TargetType distinguishes how an authorized Android target is reached.
// The assessment pipeline is transport-agnostic, but the type is recorded so
// scoping and reporting stay explicit.
type TargetType string

const (
	// TargetUSB identifies a device physically connected over ADB.
	TargetUSB TargetType = "usb"
	// TargetNetwork identifies a specific authorized Android device reached
	// by its known IP address.
	TargetNetwork TargetType = "network"
	// TargetAPK identifies an offline APK artifact supplied for static
	// analysis. It has no live device behind it.
	TargetAPK TargetType = "apk"
)

// Authorization records the explicit consent state of a target. Every
// assessment must begin with an authorized target; the framework refuses to
// proceed otherwise.
type Authorization struct {
	Granted   bool      `json:"granted"`
	GrantedAt time.Time `json:"granted_at,omitempty"`
	Scope     string    `json:"scope,omitempty"`
	GrantedBy string    `json:"granted_by,omitempty"`
	Method    string    `json:"method,omitempty"`
}

// Target is the normalized object every assessment stage operates on. It
// carries connection information, the discovered device metadata, and the
// explicit authorization gate.
type Target struct {
	ID        string        `json:"id"`
	Name      string        `json:"name,omitempty"`
	Type      TargetType    `json:"type"`
	Address   string        `json:"address,omitempty"`
	Serial    string        `json:"serial,omitempty"`
	Auth      Authorization `json:"authorization"`
	Device    *DeviceInfo   `json:"device,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	Profile   string        `json:"profile,omitempty"`
}

// Authorized reports whether the target passed the authorization gate.
func (t *Target) Authorized() bool {
	return t != nil && t.Auth.Granted
}

// DisplayName returns a short human-readable label for the target.
func (t *Target) DisplayName() string {
	if t == nil {
		return "<nil>"
	}
	switch {
	case t.Name != "":
		return t.Name
	case t.Address != "":
		return t.Address
	case t.Serial != "":
		return t.Serial
	case t.Type != "":
		return string(t.Type)
	default:
		return "unknown target"
	}
}
