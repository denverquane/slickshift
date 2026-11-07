package bot

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/denverquane/slickshift/shift"
	"github.com/gin-gonic/gin"
)

// StartAPIServer serves the API on the port. If apiKey is set, endpoints that modify data or return user data require it
func (bot *Bot) StartAPIServer(port, apiKey string) {
	err := bot.apiRouter(apiKey).Run(":" + port)
	if err != nil {
		slog.Error("API server stopped", "error", err.Error())
	}
}

// requireAPIKey rejects requests without the API key as a bearer token (like "Authorization: Bearer <key>").
// If apiKey is empty, all requests are allowed
func requireAPIKey(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey == "" {
			c.Next()
			return
		}
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing or invalid API key"})
			return
		}
		c.Next()
	}
}

func (bot *Bot) apiRouter(apiKey string) *gin.Engine {
	r := gin.Default()

	codes := r.Group("/codes", requireAPIKey(apiKey))
	{
		codes.POST("/:code", func(c *gin.Context) {
			code := c.Param("code")

			game := c.DefaultQuery("game", string(shift.Borderlands4))
			if !shift.ValidGame(game) {
				c.JSON(http.StatusBadRequest, gin.H{"message": "invalid game"})
				return
			}
			source := c.DefaultQuery("source", "")
			if !shift.CodeRegex.MatchString(code) {
				c.JSON(http.StatusBadRequest, gin.H{"message": "invalid code"})
				return
			}
			if bot.storage.CodeExists(code) {
				c.JSON(http.StatusConflict, gin.H{"message": "code already exists"})
				return
			}

			var sourceAddr *string
			if source != "" {
				sourceAddr = &source
			}

			err := bot.storage.AddCode(code, game, nil, sourceAddr)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			// trigger reprocessing because we got a new code
			bot.triggerRedemptionProcessing("")

			c.JSON(http.StatusCreated, gin.H{"code": code, "game": game, "source": source})
		})
	}
	redemptions := r.Group("/redemptions", requireAPIKey(apiKey))
	{
		redemptions.GET("/:user_id", func(c *gin.Context) {
			userID := c.Param("user_id")
			if userID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "user_id required"})
				return
			}
			_, err := strconv.ParseUint(userID, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "user_id invalid"})
				return
			}
			quantity := c.DefaultQuery("quantity", "3")
			quantityNum, err := strconv.ParseUint(quantity, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "invalid quantity"})
				return
			}

			redems, err := bot.storage.GetRecentRedemptionsForUser(userID, "", int(quantityNum))
			if err != nil {
				slog.Error("Error fetching redemptions", "user_id", userID, "error", err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"redemptions": redems})
		})
	}
	// statistics are already public through the /info command, so they don't require the API key
	info := r.Group("/info")
	{
		info.GET("", func(c *gin.Context) {
			stats, err := bot.storage.GetStatistics("")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			c.JSON(http.StatusOK, stats)
		})
	}

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, "Hello, World!")
	})

	return r
}
