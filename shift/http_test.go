package shift

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

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

func testResponse(status int, location string) http.Response {
	resp := http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("<html><body>You are being redirected.</body></html>")),
	}
	if location != "" {
		resp.Header.Set("Location", location)
	}
	return resp
}

// SHiFT redirects to the sign-in page (like this, verified against the live site) when cookies are expired or invalid
func TestReadAsHTML_SignInRedirectIsNotLoggedIn(t *testing.T) {
	_, err := readAsHTML(testResponse(http.StatusFound, HOME+"?redirect_to=https%3A%2F%2Fshift.gearboxsoftware.com%2Frewards"))
	if !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("Expected ErrNotLoggedIn, got %v", err)
	}
}

func TestReadAsHTML_OtherResponsesAreNotNotLoggedIn(t *testing.T) {
	for _, resp := range []http.Response{
		testResponse(http.StatusFound, "https://shift.gearboxsoftware.com/rewards"),
		testResponse(http.StatusServiceUnavailable, ""),
	} {
		_, err := readAsHTML(resp)
		if err == nil || errors.Is(err, ErrNotLoggedIn) {
			t.Errorf("Expected an error other than ErrNotLoggedIn for status %d, got %v", resp.StatusCode, err)
		}
	}

	doc, err := readAsHTML(testResponse(http.StatusOK, ""))
	if err != nil || doc == nil {
		t.Errorf("Expected a document for status 200, got %v", err)
	}
}
