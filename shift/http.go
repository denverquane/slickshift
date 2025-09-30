package shift

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	HOME        = "https://shift.gearboxsoftware.com/home"
	SESSIONS    = "https://shift.gearboxsoftware.com/sessions"
	REWARDS     = "https://shift.gearboxsoftware.com/rewards"
	ENTITLEMENT = "https://shift.gearboxsoftware.com/entitlement_offer_codes"
	REDEMPTIONS = "https://shift.gearboxsoftware.com/code_redemptions"
)

var GearboxURL = &url.URL{
	Scheme: "https",
	Host:   "shift.gearboxsoftware.com",
}

var defaultHeaders = http.Header{
	"User-Agent":                []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/119.0"},
	"Accept":                    []string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"},
	"Accept-Language":           []string{"en-US,en;q=0.5"},
	"Accept-Encoding":           []string{"gzip, deflate, br"},
	"DNT":                       []string{"1"},
	"Connection":                []string{"keep-alive"},
	"Upgrade-Insecure-Requests": []string{"1"},
}

func readAsHTML(resp http.Response) (*goquery.Document, error) {
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("invalid response code")
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, errors.New("invalid html")
	}

	return doc, nil
}

func readAsJson(resp http.Response) (map[string]any, error) {
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("invalid response json")
	}

	var jsonMap map[string]any
	err = json.Unmarshal(bodyBytes, &jsonMap)
	if err != nil {
		return nil, err
	}

	return jsonMap, nil
}

type httpClient struct {
	client  http.Client
	headers http.Header
}

func newHttpClient(cookies []*http.Cookie) (*httpClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.New("failed to setup cookies")
	}
	jar.SetCookies(GearboxURL, cookies)

	return &httpClient{
		http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't follow redirects automatically - we want to handle them manually
				return http.ErrUseLastResponse
			},
		},
		defaultHeaders,
	}, nil
}

func (client *httpClient) do(req *http.Request, headers map[string]string) (*http.Response, error) {
	for k, v := range client.headers {
		for _, x := range v {
			req.Header.Set(k, x)
		}
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return client.client.Do(req)
}

func setsRequiredCookies(resp *http.Response) bool {
	cookies := resp.Header.Values("Set-Cookie")

	return len(ParseRequiredCookies(cookies)) == 2
}

func ParseRequiredCookies(cookies []string) []*http.Cookie {
	var newCookies []*http.Cookie

	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if strings.HasPrefix(cookie, "_session_id=") || strings.HasPrefix(cookie, "si=") {
			first := strings.Split(cookie, ";")
			if len(first) < 1 {
				continue
			}
			items := strings.Split(first[0], "=")
			if len(items) != 2 {
				continue
			}
			newCookies = append(newCookies, &http.Cookie{Name: items[0], Value: items[1]})
		}
	}
	return newCookies
}

func (client *httpClient) Get(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.do(req, headers)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("received nil response")
	}
	return resp, nil
}

func (client *httpClient) GetAsHTML(url string, headers map[string]string) (*goquery.Document, error) {
	resp, err := client.Get(url, headers)
	if err != nil {
		return nil, err
	}
	return readAsHTML(*resp)
}

func (client *httpClient) GetAsJSON(url string, headers map[string]string) (map[string]any, error) {
	resp, err := client.Get(url, headers)
	if err != nil {
		return nil, err
	}
	return readAsJson(*resp)
}

func (client *httpClient) PostForm(url string, headers map[string]string, data string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(data))
	if err != nil {
		return nil, errors.New("failed to create login request: " + err.Error())
	}
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	return client.do(req, headers)
}
