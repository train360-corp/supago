package utils

import "testing"

func TestLogger(t *testing.T) {
	if Logger() == nil {
		t.Error("Logger is nil")
	}
}
