package stream

import (
	"testing"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
)

func TestNormalizeIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.1", "192.168.1.1"},
		{"192.168.1.1:8080", "192.168.1.1"},
		{"::1", "::1"},
		{"[::1]:80", "::1"},
		{"", ""},
		{"hostname.local", "hostname.local"},
	}

	for _, tt := range tests {
		got := normalizeIP(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeIP(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeIP_Multicast(t *testing.T) {
	got := normalizeIP("224.0.0.1")
	if got != "" {
		t.Errorf("multicast should be discarded, got %q", got)
	}
}

func TestNormalize_Event(t *testing.T) {
	ev := models.NetworkEvent{
		Source:      "10.0.0.1:443",
		Destination: "224.0.0.1",
	}
	normalized := Normalize(ev)
	if normalized.Source != "10.0.0.1" {
		t.Errorf("source should be stripped of port, got %q", normalized.Source)
	}
	if normalized.Destination != "" {
		t.Errorf("multicast destination should be empty, got %q", normalized.Destination)
	}
}
