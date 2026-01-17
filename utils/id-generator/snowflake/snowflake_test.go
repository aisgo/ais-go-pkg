package snowflake

import (
	"testing"
)

func TestNewGeneratorInvalidNodeID(t *testing.T) {
	if _, err := NewGenerator(-1); err == nil {
		t.Fatalf("expected error for negative node id")
	}
	if _, err := NewGenerator(MaxNodeID + 1); err == nil {
		t.Fatalf("expected error for too large node id")
	}
}

func TestGeneratorGenerateAndParse(t *testing.T) {
	gen, err := NewGenerator(1)
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	id := gen.Generate()
	if id == 0 {
		t.Fatalf("expected non-zero id")
	}
	if gen.GenerateString() == "" {
		t.Fatalf("expected non-empty id string")
	}
	_, nodeID := Parse(id)
	if nodeID != 1 {
		t.Fatalf("unexpected node id: %d", nodeID)
	}
}

func TestGetEnvNodeID(t *testing.T) {
	t.Setenv(EnvNodeID, "12")
	if id := getEnvNodeID(); id != 12 {
		t.Fatalf("unexpected node id: %d", id)
	}

	t.Setenv(EnvNodeID, "bad")
	if id := getEnvNodeID(); id != DefaultNodeID {
		t.Fatalf("expected default node id, got: %d", id)
	}
}
