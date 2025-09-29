package shift

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Platform string

const (
	Steam    Platform = "steam"
	Epic     Platform = "epic"
	XboxLive Platform = "xboxlive"
	PSN      Platform = "psn"
)

//type Title string
//
//const (
//	BL4 Title = "oak2"
//)

type Game string

const (
	Borderlands4 Game = "Borderlands 4"
)

type Client struct {
	hClient    httpClient
	hasCookies bool
}

func NewClient(cookies []*http.Cookie) (*Client, error) {
	hClient, err := newHttpClient(cookies)
	if err != nil {
		return nil, err
	}
	return &Client{
		*hClient,
		cookies != nil && len(cookies) > 1, // require both the si and _session_id headers
	}, nil
}

func (client *Client) Login(email string, password string) error {
	doc, err := client.hClient.GetAsHTML(HOME, nil)
	if err != nil {
		return err
	}

	csrfToken, exists := doc.Find("input[name='authenticity_token']").Attr("value")
	if !exists {
		return errors.New("failed to find authenticity token in login form")
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
		return errors.New("failed to submit login credentials: " + err.Error())
	}
	defer resp.Body.Close()

	// Check for successful login (should be a redirect)
	if resp.StatusCode == 503 {
		return errors.New("SHiFT login service is temporarily unavailable (503). This may be due to rate limiting, maintenance, or the service being overloaded. Please try again later.")
	}

	// Check for successful login - should be 302 redirect
	if resp.StatusCode == 302 {
		location := resp.Header.Get("Location")

		// Check if this is a failed login redirect (back to home with redirect_to=false)
		if bytes.Contains([]byte(location), []byte("home?redirect_to=false")) {
			return errors.New("login failed - invalid credentials (redirected back to login page)")
		}

		if location != "" {
			valid := setsRequiredCookies(resp)

			if valid {
				client.hasCookies = true
				return nil
			}
			return errors.New("redirected elsewhere; can't determine session cookies from login page")
		}
	}

	if resp.StatusCode != 302 && resp.StatusCode != 200 {
		return errors.New("unexpected login response status: " + fmt.Sprintf("%d", resp.StatusCode))
	}

	if resp.StatusCode != 302 {
		// Read the response body to get more details about the error
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)

		// Look for specific error messages in the response
		if bytes.Contains(bodyBytes, []byte("Invalid email or password")) ||
			bytes.Contains(bodyBytes, []byte("invalid email or password")) ||
			bytes.Contains(bodyBytes, []byte("Invalid credentials")) {
			return errors.New("invalid email or password - please check your credentials")
		}

		// Look for other common error patterns
		if bytes.Contains(bodyBytes, []byte("alert-danger")) ||
			bytes.Contains(bodyBytes, []byte("error-message")) ||
			bytes.Contains(bodyBytes, []byte("field_with_errors")) {
			return errors.New("login failed - form validation error or invalid credentials")
		}

		// Check if we're still on the login page (sign in form present)
		if bytes.Contains(bodyBytes, []byte("Sign in")) && bytes.Contains(bodyBytes, []byte("user[email]")) {
			return errors.New("login failed - still on login page, likely invalid credentials")
		}

		// If it's a 200 but not a redirect, it might be the login page with errors
		if resp.StatusCode == 200 {
			return errors.New("login failed - credentials may be invalid or additional verification required (status: 200, expected 302 redirect)")
		}

		maxLen := 200
		if len(bodyStr) < maxLen {
			maxLen = len(bodyStr)
		}
		return errors.New("failed to login - server error (status: " + fmt.Sprintf("%d", resp.StatusCode) + "). Expected 302 redirect for successful login. Response: " + bodyStr[:maxLen])
	}

	return errors.New("failed to extract session cookie")
}

func (client *Client) DumpCookies() []*http.Cookie {
	return client.hClient.client.Jar.Cookies(GearboxURL)
}

func (client *Client) RedeemCode(code string, platform Platform) (string, error) {
	if !client.hasCookies {
		return "", errors.New("no cookies found, login client before attempting to redeem code")
	}
	headers := map[string]string{}
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
		//"X-CSRF-TOKEN":     csrfToken,
		//"Referer":          REWARDS,
		"X-Requested-With": "XMLHttpRequest",
	}

	doc, err = client.hClient.GetAsHTML(ENTITLEMENT+"?code="+code, headers)
	if err != nil {
		return "", err
	}

	csrfToken, exists = doc.Find("input[name='authenticity_token']").Attr("value")
	if !exists {
		text := strings.TrimSpace(doc.Text())
		respType := DetermineResponseType(text)
		if respType != Unrecognized {
			return text, nil
		} else {
			log.Println("Undetected response message when authenticity token is not found:")
			log.Println(text)
		}
		return "", errors.New("failed to find authenticity token in code redemption form")
	}
	check, exists := doc.Find("input[name='archway_code_redemption[check]']").Attr("value")
	if !exists {
		return "", errors.New("failed to find archway_code_redemption[check] in form")
	}

	game, exists := doc.Find("input[name='archway_code_redemption[title]']").Attr("value")
	if !exists {
		return "", errors.New("failed to find archway_code_redemption[title] in form")
	}

	time.Sleep(1 * time.Second)

	// Prepare form data using proper URL encoding
	formValues := url.Values{}
	formValues.Set("utf8", "✓")
	formValues.Set("authenticity_token", csrfToken)
	formValues.Set("archway_code_redemption[code]", code)
	formValues.Set("archway_code_redemption[service]", string(platform))
	formValues.Set("archway_code_redemption[check]", check)
	formValues.Set("archway_code_redemption[title]", game)
	// TODO how does this work for other platforms besides Steam/PSN
	formValues.Set("commit", commitMessage(platform))

	formData := formValues.Encode()

	headers = map[string]string{
		//"X-CSRF-TOKEN": csrfToken,
		//"Referer": REWARDS,
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
				//"X-CSRF-TOKEN":     csrfToken,
				//"Referer":          REWARDS,
				"X-Requested-With": "XMLHttpRequest",
			}

			// attempt to mitigate issue where sometimes the json response is "in_progress"
			time.Sleep(1 * time.Second)

			js, err := client.hClient.GetAsJSON(location, headers)
			if err != nil {
				return "", err
			}
			//log.Println(js)
			if text, ok := js["text"]; ok {
				respType := DetermineResponseType(text.(string))
				if respType != Unrecognized {
					return text.(string), nil
				} else {
					log.Println("Undetected response message after posting code redemption")
					log.Println(text.(string))
				}
				return text.(string), nil
			} else {
				log.Println(js)
				return "", errors.New("failed to read json text status returned from code redemption")
			}
		}
	}

	return "", nil
}

func commitMessage(platform Platform) string {
	switch platform {
	case Steam:
		return "Redeem for Steam"
	case PSN:
		return "Redeem for PSN"
	case Epic:
		return "Redeem for Epic"
	case XboxLive:
		return "Redeem for Xbox Live"
	default:
		return ""
	}
}

func (client *Client) CheckRewards(platform Platform, game Game, limit int) ([]Reward, error) {
	if !client.hasCookies {
		return nil, errors.New("no cookies found, login client before attempting to load rewards")
	}
	headers := map[string]string{}
	doc, err := client.hClient.GetAsHTML(REWARDS, headers)
	if err != nil {
		return nil, err
	}

	var rewards []Reward
	currentGame := ""
	selector := fmt.Sprintf("div.tab-pane.well#%s div.sh_reward_list", string(platform))
	doc.Find(selector).Children().Each(func(i int, s *goquery.Selection) {
		if s.HasClass("shift-secondary-title") {
			// Update current game context
			currentGame = strings.TrimSpace(s.Find("h2").Text())
		} else if goquery.NodeName(s) == "dl" && currentGame == string(game) {
			// Parse a reward
			title := strings.TrimSpace(s.Find("dt").Text())
			date := strings.TrimSpace(s.Find("dd .reward_unlocked").Text())

			dd := s.Find("dd").Clone()
			dd.Find(".reward_unlocked").Remove()
			description := strings.TrimSpace(dd.Text())
			rewards = append(rewards, Reward{
				Title:       title,
				Date:        date,
				Description: description,
			})
			if limit > 0 && len(rewards) == limit {
				return
			}
		}
	})
	return rewards, nil
}
