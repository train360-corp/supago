package postgres

import "testing"

func TestGetEmbeddedFiles(t *testing.T) {
	if len(GetEmbeddedFiles()) != 7 {
		t.Error("Expected 7 files")
	}
}
