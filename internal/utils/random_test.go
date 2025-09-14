package utils

import "testing"

func TestRandomString(t *testing.T) {
	if len(RandomString(6)) != 6 {
		t.Errorf("Random string length is incorrect")
	}
}
