package bot

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

const testAPIKey = "test_api_key"

func newTestRouter(t *testing.T, apiKey string) (*gin.Engine, *Bot) {
	gin.SetMode(gin.TestMode)
	bot, _ := newTestBot(t, nil)
	return bot.apiRouter(apiKey), bot
}

func serve(r *gin.Engine, method, path, authorization string) int {
	req := httptest.NewRequest(method, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// API_KEY is opt-in, so existing setups without it keep working
func TestAPI_NoKeyConfiguredAllowsRequests(t *testing.T) {
	r, bot := newTestRouter(t, "")

	if code := serve(r, http.MethodPost, "/codes/"+testCode, ""); code != http.StatusCreated {
		t.Errorf("Adding a code should succeed, got %d", code)
	}
	if len(bot.redemptionTrigger) != 1 {
		t.Errorf("Adding a code should trigger redemption processing")
	}
	if code := serve(r, http.MethodGet, "/redemptions/123", ""); code != http.StatusOK {
		t.Errorf("Getting redemptions should succeed, got %d", code)
	}
}

func TestAPI_KeyRequired(t *testing.T) {
	r, _ := newTestRouter(t, testAPIKey)

	for _, authorization := range []string{"", "Bearer wrong_key", testAPIKey, "Basic " + testAPIKey} {
		if code := serve(r, http.MethodPost, "/codes/"+testCode, authorization); code != http.StatusUnauthorized {
			t.Errorf("Adding a code with Authorization %q should be unauthorized, got %d", authorization, code)
		}
		if code := serve(r, http.MethodGet, "/redemptions/123", authorization); code != http.StatusUnauthorized {
			t.Errorf("Getting redemptions with Authorization %q should be unauthorized, got %d", authorization, code)
		}
	}

	if code := serve(r, http.MethodPost, "/codes/"+testCode, "Bearer "+testAPIKey); code != http.StatusCreated {
		t.Errorf("Adding a code with the API key should succeed, got %d", code)
	}
	if code := serve(r, http.MethodGet, "/redemptions/123", "Bearer "+testAPIKey); code != http.StatusOK {
		t.Errorf("Getting redemptions with the API key should succeed, got %d", code)
	}
}

// statistics are already public through the /info command
func TestAPI_InfoIsPublic(t *testing.T) {
	r, _ := newTestRouter(t, testAPIKey)

	if code := serve(r, http.MethodGet, "/info", ""); code != http.StatusOK {
		t.Errorf("Getting info without the API key should succeed, got %d", code)
	}
}
