package auth

import "testing"

func TestSliceMatches(t *testing.T) {
	requested := []string{"a", "b", "c"}
	allowed := []string{"a", "c"}
	if _, mismatch := SliceMatches(requested, allowed); mismatch != "b" {
		t.Errorf("expected to fail with missmatch 'b', got: %s", mismatch)
	}

	requested = []string{"a", "c"}
	if ok, mismatch := SliceMatches(requested, allowed); !ok {
		t.Errorf("expected to succeed with no missmatch, got: %s", mismatch)
	}
}
