package main

import (
	"encoding/base64"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/denverquane/slickshift/bot"
	"github.com/denverquane/slickshift/data"
	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"
)

// inject with -ldflags when compiling, like
// -ldflags "-X main.Version=$(git describe --tags "$(git rev-list --tags --max-count=1)" 2>/dev/null || echo 'dev')
// -X main.Commit=$(git rev-parse --short HEAD)"
var (
	Version string
	Commit  string
)

func main() {
	secretKey := os.Getenv("ENCRYPTION_KEY_B64")
	token := os.Getenv("DISCORD_BOT_TOKEN")
	guildID := os.Getenv("DISCORD_GUILD_ID")
	redeemInterval := os.Getenv("REDEEM_INTERVAL")
	if redeemInterval == "" {
		redeemInterval = "5"
		slog.Info("No REDEEM_INTERVAL set, defaulting to " + redeemInterval)
	}
	redeemIntervalInt, err := strconv.Atoi(redeemInterval)
	if err != nil {
		log.Fatalf("Error parsing REDEEM_INTERVAL: %v", err)
	} else if redeemIntervalInt < 1 {
		log.Fatalf("REDEEM_INTERVAL cannot be less than 1")
	}
	dbFilePath := os.Getenv("DATABASE_FILE_PATH")
	if dbFilePath == "" {
		dbFilePath = "./sqlite.db"
		slog.Info("Database file path not set, defaulting to " + dbFilePath)
	}
	if secretKey == "" {
		log.Fatal("ENCRYPTION_KEY_B64 environment variable not set")
	}
	secretKeyBytes, err := base64.StdEncoding.DecodeString(secretKey)
	if err != nil {
		log.Fatal("Error decoding secret key", err.Error())
	}

	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN environment variable not set")
	}
	slog.Info("Env",
		"Version", Version,
		"Commit", Commit,
		"DATABASE_FILE_PATH", dbFilePath,
		"REDEEM_INTERVAL", redeemIntervalInt,
		"DISCORD_GUILD_ID", guildID,
		"DISCORD_BOT_TOKEN", "<redacted>",
		"ENCRYPTION_KEY_B64", "<redacted>",
	)

	encryptor, err := store.NewEncryptor(secretKeyBytes)
	if err != nil {
		log.Fatal(err)
	}

	storage, err := store.NewSqliteStore(dbFilePath, encryptor)
	if err != nil {
		log.Fatal(err)
	}

	codes := data.DefaultBL4Codes()
	for code := range codes {
		err = storage.AddCode(code, string(shift.Borderlands4), nil, nil)
		if err != nil {
			slog.Error("error adding default code", "error", err.Error())
		}
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	b, err := bot.CreateNewBot(token, storage, Version, Commit)
	if err != nil {
		log.Fatal(err)
	}
	err = b.Start()
	if err != nil {
		log.Fatal(err)
	}

	go b.StartAPIServer("8080")

	cmds, err := b.RegisterCommands(guildID)
	if err != nil {
		log.Fatal(err)
	}

	kill := make(chan bool, 1)

	go b.StartUserRedemptionProcessing(time.Minute*time.Duration(redeemIntervalInt), kill)

	<-sc
	log.Printf("Received Sigterm or Kill signal. Bot terminating after deleting commands")
	kill <- true

	b.DeleteCommands(guildID, cmds)
	err = b.Stop()
	if err != nil {
		log.Fatal(err)
	}
}
