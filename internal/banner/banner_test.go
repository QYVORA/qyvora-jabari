package banner

import (
	"strings"
	"testing"
)

// TestArtNonEmpty guards against the banner being accidentally emptied by a
// bad regeneration step.
func TestArtNonEmpty(t *testing.T) {
	if strings.TrimSpace(Art) == "" {
		t.Fatal("Art is empty")
	}
}

// TestArtGlyphs checks that every glyph of the brand palette is present, so
// the console can map characters to the logo colors with confidence.
func TestArtGlyphs(t *testing.T) {
	for _, g := range []string{"%", "#", "+", "*"} {
		if !strings.Contains(Art, g) {
			t.Errorf("Art missing brand glyph %q", g)
		}
	}
}

// TestArtLineWidths sanity-checks the art geometry: the canonical robot mark
// is roughly square (about 25 lines tall, 40-64 columns wide). Wide
// deviations indicate the wrong file was embedded.
func TestArtLineWidths(t *testing.T) {
	lines := strings.Split(Art, "\n")
	nonEmpty := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonEmpty++
		if w := len([]rune(line)); w < 40 || w > 70 {
			t.Errorf("art line %q has width %d, want 40-70", line, w)
		}
	}
	if nonEmpty != 25 {
		t.Errorf("art has %d non-empty lines, want 25", nonEmpty)
	}
}
