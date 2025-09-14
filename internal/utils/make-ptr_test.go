package utils

import "testing"

func TestPointer(t *testing.T) {
	got := Pointer("foo")
	if "foo" != *got {
		t.Errorf("Pointer(foo) = %v; want foo", got)
	}
}
