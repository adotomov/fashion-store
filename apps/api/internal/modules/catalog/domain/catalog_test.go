package domain

import "testing"

func TestStatus_Valid(t *testing.T) {
	valid := []Status{StatusDraft, StatusActive, StatusDisabled}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("expected %q to be valid", s)
		}
	}

	if Status("archived").Valid() {
		t.Error("expected unknown status to be invalid")
	}
}
