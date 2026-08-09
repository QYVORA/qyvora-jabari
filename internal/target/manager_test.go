package target

import (
	"testing"

	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

func authorizedUSB() *models.Target {
	return &models.Target{
		ID:     "tgt-1",
		Name:   "USB device abc",
		Type:   models.TargetUSB,
		Serial: "abc",
		Auth:   models.Authorization{Granted: true, Scope: "test"},
	}
}

func TestSetRequiresAuthorization(t *testing.T) {
	m := NewManager()
	unauthorized := &models.Target{
		ID:     "tgt-x",
		Type:   models.TargetUSB,
		Serial: "abc",
	}
	if err := m.Set(unauthorized); err == nil {
		t.Fatal("Set should reject an unauthorized target")
	}
	if m.Current() != nil {
		t.Error("rejected target must not become current")
	}
}

func TestSetAndGet(t *testing.T) {
	m := NewManager()
	if err := m.Set(authorizedUSB()); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if m.Current().ID != "tgt-1" {
		t.Errorf("Current = %s, want tgt-1", m.Current().ID)
	}
	if got, ok := m.Get("tgt-1"); !ok || got != m.Current() {
		t.Error("Get(tgt-1) did not return the current target")
	}
}

func TestSetAssignsID(t *testing.T) {
	m := NewManager()
	tg := &models.Target{Type: models.TargetUSB, Serial: "abc", Auth: models.Authorization{Granted: true}}
	if err := m.Set(tg); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if tg.ID == "" {
		t.Error("Set should assign an ID when one is missing")
	}
}

func TestList(t *testing.T) {
	m := NewManager()
	_ = m.Set(authorizedUSB())
	second := authorizedUSB()
	second.ID = "tgt-2"
	_ = m.Set(second)
	if got := len(m.List()); got != 2 {
		t.Errorf("List returned %d targets, want 2", got)
	}
}
