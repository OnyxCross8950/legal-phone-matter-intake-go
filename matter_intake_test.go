package main

import (
	"testing"
	"time"
)

func TestMatterAdmission(t *testing.T) {
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	base := matter{Phone: "+14155550123", MatterID: "case-42", SignedDocument: "signed-retainer-42", Deadline: "2026-09-21"}
	tests := []struct {
		name     string
		change   func(*matter)
		accepted bool
	}{
		{"signed retainer and upcoming deadline", func(*matter) {}, true},
		{"expired deadline", func(m *matter) { m.Deadline = "2026-09-19" }, false},
		{"missing signed document", func(m *matter) { m.SignedDocument = "" }, false},
		{"non international phone", func(m *matter) { m.Phone = "4155550123" }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := base
			tc.change(&m)
			if got := validateMatter(m, now) == nil; got != tc.accepted {
				t.Fatalf("accepted = %v, want %v", got, tc.accepted)
			}
		})
	}
}
