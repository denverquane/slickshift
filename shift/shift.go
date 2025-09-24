package shift

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

const (
	HOME        = "https://shift.gearboxsoftware.com/home"
	SESSIONS    = "https://shift.gearboxsoftware.com/sessions"
	REWARDS     = "https://shift.gearboxsoftware.com/rewards"
	ENTITLEMENT = "https://shift.gearboxsoftware.com/entitlement_offer_codes"
	REDEMPTIONS = "https://shift.gearboxsoftware.com/code_redemptions"
)

type Client struct {
	hClient httpClient
}

func NewClient() (*Client, error) {
	hClient, err := newHttpClient()
	if err != nil {
		return nil, err
	}
	return &Client{
		*hClient,
	}, nil
}

func (client *Client) Login(email string, password string) (string, error) {
	doc, err := client.hClient.GetAsHTML(HOME, nil)
	if err != nil {
		return "", err
	}

	csrfToken, exists := doc.Find("input[name='authenticity_token']").Attr("value")
	if !exists {
		return "", errors.New("failed to find authenticity token in login form")
	}

	// Add a small delay to mimic human behavior
	time.Sleep(1 * time.Second)

	// Prepare form data using proper URL encoding
	formValues := url.Values{}
	formValues.Set("utf8", "✓")
	formValues.Set("authenticity_token", csrfToken)
	formValues.Set("user[email]", email)
	formValues.Set("user[password]", password)
	formValues.Set("commit", "SIGN IN")

	formData := formValues.Encode()

	headers := map[string]string{
		"Referer": HOME,
	}
	resp, err := client.hClient.PostForm(SESSIONS, headers, formData)
	if err != nil {
		return "", errors.New("failed to submit login credentials: " + err.Error())
	}
	defer resp.Body.Close()

	// Check for successful login (should be a redirect)
	if resp.StatusCode == 503 {
		return "", errors.New("SHiFT login service is temporarily unavailable (503). This may be due to rate limiting, maintenance, or the service being overloaded. Please try again later.")
	}

	// Check for successful login - should be 302 redirect
	if resp.StatusCode == 302 {
		location := resp.Header.Get("Location")

		// Check if this is a failed login redirect (back to home with redirect_to=false)
		if bytes.Contains([]byte(location), []byte("home?redirect_to=false")) {
			return "", errors.New("login failed - invalid credentials (redirected back to login page)")
		}

		if location != "" {
			sessionCookie := extractSessionID(resp)

			if sessionCookie != "" {
				return sessionCookie, nil
			}
			return "", errors.New("redirected elsewhere; can't determine session cookie from login page")
		}
	}

	if resp.StatusCode != 302 && resp.StatusCode != 200 {
		return "", errors.New("unexpected login response status: " + fmt.Sprintf("%d", resp.StatusCode))
	}

	if resp.StatusCode != 302 {
		// Read the response body to get more details about the error
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)

		// Look for specific error messages in the response
		if bytes.Contains(bodyBytes, []byte("Invalid email or password")) ||
			bytes.Contains(bodyBytes, []byte("invalid email or password")) ||
			bytes.Contains(bodyBytes, []byte("Invalid credentials")) {
			return "", errors.New("invalid email or password - please check your credentials")
		}

		// Look for other common error patterns
		if bytes.Contains(bodyBytes, []byte("alert-danger")) ||
			bytes.Contains(bodyBytes, []byte("error-message")) ||
			bytes.Contains(bodyBytes, []byte("field_with_errors")) {
			return "", errors.New("login failed - form validation error or invalid credentials")
		}

		// Check if we're still on the login page (sign in form present)
		if bytes.Contains(bodyBytes, []byte("Sign in")) && bytes.Contains(bodyBytes, []byte("user[email]")) {
			return "", errors.New("login failed - still on login page, likely invalid credentials")
		}

		// If it's a 200 but not a redirect, it might be the login page with errors
		if resp.StatusCode == 200 {
			return "", errors.New("login failed - credentials may be invalid or additional verification required (status: 200, expected 302 redirect)")
		}

		maxLen := 200
		if len(bodyStr) < maxLen {
			maxLen = len(bodyStr)
		}
		return "", errors.New("failed to login - server error (status: " + fmt.Sprintf("%d", resp.StatusCode) + "). Expected 302 redirect for successful login. Response: " + bodyStr[:maxLen])
	}

	return "", errors.New("failed to extract session cookie")
}

func (client *Client) RedeemCode(cookie, code, platform string) (string, error) {
	headers := map[string]string{
		"Cookie": cookie,
	}
	doc, err := client.hClient.GetAsHTML(REWARDS, headers)
	if err != nil {
		return "", err
	}

	csrfToken, exists := doc.Find("meta[name='csrf-token']").Attr("content")
	if !exists {
		return "", errors.New("failed to find csrf token in redemption form")
	}

	// override all headers to be clear about what's required/expected
	headers = map[string]string{
		"Cookie":           cookie,
		"X-CSRF-TOKEN":     csrfToken,
		"Referer":          REWARDS,
		"X-Requested-With": "XMLHttpRequest",
	}

	doc, err = client.hClient.GetAsHTML(ENTITLEMENT+"?code="+code, headers)
	if err != nil {
		return "", err
	}

	csrfToken, exists = doc.Find("input[name='authenticity_token']").Attr("value")
	if !exists {
		if strings.TrimSpace(doc.Text()) == NOT_EXIST {
			return NOT_EXIST, nil // maybe should return an error, but it's a case we can catch, so handle it elsewhere
		}
		return "", errors.New("failed to find authenticity token in code redemption form")
	}
	check, exists := doc.Find("input[name='archway_code_redemption[check]']").Attr("value")
	if !exists {
		return "", errors.New("failed to find archway_code_redemption[check] in form")
	}

	time.Sleep(1 * time.Second)

	// Prepare form data using proper URL encoding
	formValues := url.Values{}
	formValues.Set("utf8", "✓")
	formValues.Set("authenticity_token", csrfToken)
	formValues.Set("archway_code_redemption[code]", code)
	formValues.Set("archway_code_redemption[service]", platform)
	formValues.Set("archway_code_redemption[check]", check)
	formValues.Set("archway_code_redemption[title]", "oak2")
	formValues.Set("commit", "Redeem for Steam")

	formData := formValues.Encode()

	headers = map[string]string{
		"Cookie":       cookie,
		"X-CSRF-TOKEN": csrfToken,
		"Referer":      REWARDS,
	}
	resp, err := client.hClient.PostForm(REDEMPTIONS, headers, formData)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == 302 {
		location := resp.Header.Get("Location")

		if bytes.Contains([]byte(location), []byte("?redirect_to=false")) {
			return "", errors.New("redeem code failed (redirected back to redeem page)")
		}

		// If it's a redirect to somewhere else, it's likely successful
		if location != "" {
			headers = map[string]string{
				"Cookie":           cookie,
				"X-CSRF-TOKEN":     csrfToken,
				"Referer":          REWARDS,
				"X-Requested-With": "XMLHttpRequest",
			}

			js, err := client.hClient.GetAsJSON(location, headers)
			if err != nil {
				return "", err
			}
			//log.Println(js)
			if text, ok := js["text"]; ok {
				return text.(string), nil
			} else {
				return "", errors.New("failed to read json text status returned from code redemption")
			}
		}
	}

	return "", nil
}
