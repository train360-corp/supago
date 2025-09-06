package services

import "testing"

func TestNewRunner(t *testing.T) {
	if _, err := NewRunner("go-test-impl"); err != nil {
		t.Errorf("error creating runner: %v", err)
	}
}
