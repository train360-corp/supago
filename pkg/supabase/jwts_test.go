package supabase

import (
	"testing"
)

func TestGetJwts(t *testing.T) {

	secret := "super-secret-password"
	expectedPublic := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjIzNjY4NDE2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoic3VwYWJhc2UiLCJyb2xlIjoiYW5vbiJ9.zzm-F12eAnEb3H1ViVK2A8EHGZLvvghe9vZmvPezdtc"
	expectedPrivate := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjIzNjY4NDE2MDAsImlhdCI6MTczNTY4OTYwMCwiaXNzIjoic3VwYWJhc2UiLCJyb2xlIjoic2VydmljZV9yb2xlIn0.AwDcdVx7oGXHy2ZUdGI6QjH1MzZrxf5NLyDxu4J-Z5g"

	jwts, err := GetJwts(secret)
	if err != nil {
		t.Fatal(err)
	}

	if jwts.Public != expectedPublic {
		t.Fatalf("expected[%s] != actual[%s]", expectedPublic, jwts.Public)
	}

	if jwts.Private != expectedPrivate {
		t.Fatalf("expected[%s] != actual[%s]", expectedPrivate, jwts.Private)
	}
}
