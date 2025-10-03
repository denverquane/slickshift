package bot

import (
	"net/http"

	"github.com/denverquane/slickshift/shift"
	"github.com/gin-gonic/gin"
)

func (bot *Bot) StartAPIServer(port string) {
	r := gin.Default()

	r.Group("/codes", func(c *gin.Context) {
		r.POST("/:code", func(c *gin.Context) {
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
			c.JSON(http.StatusCreated, gin.H{"code": code, "game": game, "source": source})
		})
	})

	r.Run(":" + port)
}
