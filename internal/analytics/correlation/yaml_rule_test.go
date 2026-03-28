package correlation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
)

func TestYAMLRule_PortScanFlood(t *testing.T) {
	rule := &YAMLRule{spec: YAMLRuleSpec{
		RuleName:  "PORT_SCAN_FLOOD",
		Severity:  "HIGH",
		EventType: "PORT_SCAN",
		Logic:     "and",
		Conditions: []Condition{
			{Field: "stream.count_1m", Operator: "gt", Value: "50"},
		},
	}}

	// Below threshold — no alert
	ev := models.NetworkEvent{
		ID:   "1",
		Type: models.EventPortScan,
		Metadata: map[string]any{
			"stream.count_1m": 30,
		},
		Timestamp: time.Now(),
	}
	if alert := rule.Evaluate(ev); alert != nil {
		t.Fatal("expected no alert for count below threshold")
	}

	// Above threshold — should alert
	ev.Metadata["stream.count_1m"] = 60
	if alert := rule.Evaluate(ev); alert == nil {
		t.Fatal("expected alert for count above threshold")
	}
}

func TestYAMLRule_OrLogic(t *testing.T) {
	rule := &YAMLRule{spec: YAMLRuleSpec{
		RuleName:  "LATERAL_MOVEMENT",
		Severity:  "HIGH",
		EventType: "PORT_SCAN",
		Logic:     "or",
		Conditions: []Condition{
			{Field: "destination", Operator: "contains", Value: "192.168."},
			{Field: "destination", Operator: "contains", Value: "10."},
		},
	}}

	ev := models.NetworkEvent{
		ID:          "1",
		Type:        models.EventPortScan,
		Destination: "192.168.1.50",
		Timestamp:   time.Now(),
	}
	if alert := rule.Evaluate(ev); alert == nil {
		t.Fatal("expected alert for internal IP target")
	}

	ev.Destination = "8.8.8.8"
	if alert := rule.Evaluate(ev); alert != nil {
		t.Fatal("expected no alert for external IP")
	}
}

func TestYAMLRule_WrongEventType(t *testing.T) {
	rule := &YAMLRule{spec: YAMLRuleSpec{
		RuleName:  "DNS_ONLY",
		EventType: "DNS",
		Logic:     "and",
		Conditions: []Condition{
			{Field: "name", Operator: "exists"},
		},
	}}

	ev := models.NetworkEvent{
		ID:   "1",
		Type: models.EventARP,
		Metadata: map[string]any{
			"name": "test",
		},
	}
	if alert := rule.Evaluate(ev); alert != nil {
		t.Fatal("expected no alert for wrong event type")
	}
}

func TestYAMLRule_EqualsOperator(t *testing.T) {
	rule := &YAMLRule{spec: YAMLRuleSpec{
		RuleName: "TEST_EQUALS",
		Logic:    "and",
		Conditions: []Condition{
			{Field: "privileged", Operator: "equals", Value: "true"},
		},
	}}

	ev := models.NetworkEvent{
		ID:       "1",
		Type:     models.EventVM,
		Metadata: map[string]any{"privileged": "true"},
	}
	if alert := rule.Evaluate(ev); alert == nil {
		t.Fatal("expected alert for equals match")
	}

	ev.Metadata["privileged"] = "false"
	if alert := rule.Evaluate(ev); alert != nil {
		t.Fatal("expected no alert for non-match")
	}
}

func TestLoadRulesFromDir(t *testing.T) {
	dir := t.TempDir()

	// Write a valid rule
	data := []byte(`name: "TEST_RULE"
description: "A test rule"
severity: "LOW"
event_type: "DNS"
logic: "and"
conditions:
  - field: "name"
    operator: "contains"
    value: "test"
`)
	if err := os.WriteFile(filepath.Join(dir, "test.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Write a non-YAML file (should be skipped)
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a rule"), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadRulesFromDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Name() != "TEST_RULE" {
		t.Fatalf("expected TEST_RULE, got %s", rules[0].Name())
	}
}

func TestLoadRulesFromDir_NonExistent(t *testing.T) {
	rules, err := LoadRulesFromDir("/nonexistent/path")
	if err != nil {
		t.Fatalf("non-existent dir should return nil error, got: %v", err)
	}
	if rules != nil {
		t.Fatal("expected nil rules for non-existent dir")
	}
}
