package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Evidence is a single reproducible artifact that supports a finding. Each
// evidence item records what was collected, from where, and when so that
// findings can be audited after the fact.
//
// The framework prefers structured evidence (configuration values, manifest
// entries, service banners) over raw dumps, and never stores credentials or
// personal data unless the session explicitly captures them as evidence.
type Evidence struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Source    string    `json:"source"`
	Content   string    `json:"content,omitempty"`
	Data      []byte    `json:"-"`
	MimeType  string    `json:"mime_type,omitempty"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
}

// Kind values for Evidence.Kind. New collectors should reuse these where
// possible so reporting stays consistent.
const (
	KindDevice        = "device"
	KindApplication   = "application"
	KindManifest      = "manifest"
	KindConfiguration = "configuration"
	KindService       = "service"
	KindLog           = "log"
	KindScreenshot    = "screenshot"
	KindArtifact      = "artifact"
	KindCommand       = "command"
)

// EvidenceID generates a short unique identifier for an evidence item.
func EvidenceID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// HashContent returns the lowercase hex SHA-256 digest of b. Every payload
// logged by the framework is hashed this way so artifacts can be tied to a
// finding without embedding the artifact itself.
func HashContent(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
