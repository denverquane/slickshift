package shift

import "testing"

func TestParseRequiredCookies(t *testing.T) {
	cookies := []string{
		"si=si_here; path=/; expires=Wed, 30 Sep 2026 00:24:22 GMT; HttpOnly",
		"_session_id=session_id_here; path=/; HttpOnly",
	}

	newCookies := ParseRequiredCookies(cookies)

	if len(newCookies) != len(cookies) {
		t.Fatal("Cookies length mismatch")
	}

	if newCookies[0].Name != "si" {
		t.Fatalf("Cookie si name mismatch: %s", newCookies[0].Name)
	}
	if newCookies[0].Value != "si_here" {
		t.Fatalf("Cookie si value mismatch: %s", newCookies[0].Value)
	}
	if newCookies[1].Name != "_session_id" {
		t.Fatalf("Cookie _session_id name mismatch: %s", newCookies[1].Name)
	}
	if newCookies[1].Value != "session_id_here" {
		t.Fatalf("Cookie _session_id value mismatch: %s", newCookies[1].Value)
	}
}
