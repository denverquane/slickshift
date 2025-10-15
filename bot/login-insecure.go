package bot

import (
	"log"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/denverquane/slickshift/shift"
)

func (bot *Bot) loginResponse(userID string, s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	email := i.ApplicationCommandData().Options[0].StringValue()
	password := i.ApplicationCommandData().Options[1].StringValue()

	client, err := shift.NewClient(nil)
	if err != nil {
		log.Println(err)
		return privateMessageResponse("I encountered an error creating an HTTP client for login. Please try again later.")
	}

	err = client.Login(email, password)
	if err != nil {
		log.Println(err)
		return privateMessageResponse("I wasn't able to log you in to SHiFT. Are you sure you provided the right credentials?")
	}
	cookies := client.DumpCookies()
	err = bot.storage.EncryptAndSetUserCookies(userID, cookies)
	if err != nil {
		log.Println(err)
		return privateMessageResponse("I logged into SHiFT with your info, but I wasn't able to store your session cookies for later...")
	}
	err = bot.storage.ClearShiftErrors(userID)
	if err != nil {
		slog.Error("Couldn't clear shift_errors for user", "user_id", userID, "error", err.Error())
	}
	bot.triggerRedemptionProcessing(userID)
	return privateMessageResponse(Cheer + " Success! " + Cheer + "\n\nI've securely stored your session cookies (and purged your email/password) for automatic SHiFT code redemption!")
}
