package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"
	"log"
)

func (bot *Bot) addResponse(userID string, s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	code := i.ApplicationCommandData().Options[0].StringValue()
	if !shift.CodeRegex.MatchString(code) {
		return privateMessageResponse("Hm, doesn't look like you provided a valid SHiFT code. It should look something like:\n\n" +
			"`XXXX-XXXXX-XXXXX-XXXXX-XXXXX`")

	}
	if bot.storage.CodeExists(code) {
		return privateMessageResponse("It looks like that code already exists!\nThanks anyways!")
	}
	var src = store.DiscordSource
	err := bot.storage.AddCode(code, string(shift.Borderlands4), &userID, &src)
	if err != nil {
		log.Println(err)
		return nil
	}
	// trigger reprocessing because a new code was added
	bot.triggerRedemptionProcessing("")

	return privateMessageResponse("Nice, thanks for adding the code! It should be tested and validated soon!")
}
