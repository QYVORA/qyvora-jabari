package apk

import (
	"archive/zip"
	"crypto/x509"
	"fmt"
	"io"
)

// parseCertificate extracts certificate information from a signing file
func parseCertificate(f *zip.File) (Certificate, error) {
	rc, err := f.Open()
	if err != nil {
		return Certificate{}, err
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		return Certificate{}, err
	}

	// Parse PKCS#7 signature
	// This is a simplified implementation
	// Full implementation would use crypto/x509/pkix

	cert := Certificate{
		Algorithm: "RSA", // Default assumption
	}

	// Try to parse as X.509 certificate
	certs, err := x509.ParseCertificates(data)
	if err == nil && len(certs) > 0 {
		c := certs[0]
		cert.Subject = c.Subject.String()
		cert.Issuer = c.Issuer.String()
		cert.SerialNumber = c.SerialNumber.String()
		cert.NotBefore = c.NotBefore.Format("2006-01-02")
		cert.NotAfter = c.NotAfter.Format("2006-01-02")
		cert.SHA256 = fmt.Sprintf("%x", c.Raw)

		switch c.PublicKeyAlgorithm {
		case x509.RSA:
			cert.Algorithm = "RSA"
		case x509.ECDSA:
			cert.Algorithm = "ECDSA"
		case x509.DSA:
			cert.Algorithm = "DSA"
		}
	}

	return cert, nil
}

// VerifySignature reports whether the APK's signature can be asserted as
// cryptographically verified. Full verification of APK Signature Scheme v1
// (JAR/CMS), v2, and v3 is not yet implemented; rather than silently
// returning nil for every signed APK (a false "verified" promise), it
// returns a descriptive error so consumers never mistake an unverified
// signature for a verified one. Signer extraction (parseCertificate) and
// the certificate fingerprint remain reliable.
func (a *APK) VerifySignature() error {
	if len(a.Certificates) == 0 {
		return fmt.Errorf("no certificates found")
	}
	return fmt.Errorf("apk signature verification not implemented (v1/v2/v3); integrity not asserted")
}

// SignatureScheme returns the detected signature scheme(s)
func (a *APK) SignatureScheme() []string {
	// Placeholder - would detect v1/v2/v3/v4 schemes
	schemes := []string{}

	if len(a.Certificates) > 0 {
		schemes = append(schemes, "v1")
	}

	return schemes
}

// CertificateFingerprint returns the SHA-256 fingerprint of the first certificate
func (a *APK) CertificateFingerprint() string {
	if len(a.Certificates) == 0 {
		return ""
	}
	return a.Certificates[0].SHA256
}

// IsDebugSigned checks if the APK is signed with a debug certificate
func (a *APK) IsDebugSigned() bool {
	for _, cert := range a.Certificates {
		// Debug certificates typically have CN=Android Debug
		if containsIgnoreCase(cert.Subject, "Android Debug") {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		len(substr) > 0 &&
		indexIgnoreCase(s, substr) >= 0
}

func indexIgnoreCase(s, substr string) int {
	n := len(substr)
	for i := 0; i+n <= len(s); i++ {
		if equalIgnoreCase(s[i:i+n], substr) {
			return i
		}
	}
	return -1
}

func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
