package utils

import "testing"

func TestShortStrLt8(t *testing.T) {
	expect := "abc123"
	got := ShortStr(expect)
	if expect != got {
		t.Errorf("expect: %s, got: %s", expect, got)
	}
}

func TestShortStrEt8(t *testing.T) {
	expect := "abcd1234"
	got := ShortStr(expect)
	if expect != got {
		t.Errorf("expect: %s, got: %s", expect, got)
	}
}

func TestShortStrGt8(t *testing.T) {
	expect := "abcd...2345"
	got := ShortStr("abcd12345")
	if expect != got {
		t.Errorf("expect: %s, got: %s", expect, got)
	}
}
