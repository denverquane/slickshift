package bot

import (
	"github.com/bwmarrin/discordgo"
	"log"
)

const settingsSuffix = "* Do you want me to message you when I redeem codes for you, or when your login details expire?\n" +
	"* Also, can you tell me what platform you'd like to auto-redeem SHiFT codes for?\n"

func (bot *Bot) settingsResponse(userID string, s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	platform, shouldDM, err := bot.storage.GetUserPlatformAndDM(userID)
	if err != nil {
		log.Println(err)
		return privateMessageResponse("Hm, I got an error fetching your platform. Please try again later.")
	}
	msg := privateMessageResponse(settingsSuffix)
	msg.Data.Components = []discordgo.MessageComponent{
		getDMComponents(true, shouldDM),
		getPlatformComponents(platform != "", platform),
	}
	return msg
}
