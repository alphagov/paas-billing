package auth

import "testing"

func TestSliceMatches(t *testing.T) {
	requested := []string{"a", "b", "c"}
	allowed := []string{"a", "c"}
	if _, missmatch := SliceMatches(requested, allowed); missmatch != "b" {
		t.Errorf("expected to fail with missmatch 'b', got: %s", missmatch)
	}

	requested = []string{"a", "c"}
	if ok, missmatch := SliceMatches(requested, allowed); !ok {
		t.Errorf("expected to succeed with no missmatch, got: %s", missmatch)
	}
}
