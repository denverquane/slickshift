package main

import (
	"github.com/denverquane/slickshift/bot"
	"github.com/denverquane/slickshift/store"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	guildID := os.Getenv("DISCORD_GUILD_ID")

	if token == "" {
		log.Fatalf("DISCORD_BOT_TOKEN environment variable not set")
	}

	storage, err := store.NewSqliteStore("./sqlite.db")
	if err != nil {
		log.Fatal(err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	b, err := bot.CreateNewBot(token, storage)
	if err != nil {
		log.Fatal(err)
	}
	err = b.Start()
	if err != nil {
		log.Fatal(err)
	}

	cmds, err := b.RegisterCommands(guildID)
	if err != nil {
		log.Fatal(err)
	}

	<-sc
	log.Printf("Received Sigterm or Kill signal. Bot will terminate in 1 second")

	b.DeleteCommands(guildID, cmds)
	err = b.Stop()
	if err != nil {
		log.Fatal(err)
	}

	//r := gin.Default()
	//
	//// Define a simple GET endpoint
	//r.GET("/ping", func(c *gin.Context) {
	//	// Return JSON response
	//	c.JSON(http.StatusOK, gin.H{
	//		"message": "pong",
	//	})
	//})
	//
	//// Start server on port 8080 (default)
	//// Server will listen on 0.0.0.0:8080 (localhost:8080 on Windows)
	//r.Run()
}
