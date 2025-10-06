package bot

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const settingsSuffix = "* Do you want me to message you when I redeem codes for you, or when your login details expire?\n" +
	"* Also, can you tell me what platform you'd like to auto-redeem SHiFT codes for?\n"

func (bot *Bot) settingsResponse(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	userID := i.Member.User.ID
	platform, shouldDM, err := bot.storage.GetUserPlatformAndDM(userID)
	if err != nil {
		log.Println(err)
		return privateMessageResponse("Hm, I got an error fetching your platform. Please try again later.")
	}
	var content string
	if platform != "" {
		content = "Your current platform for SHiFT code auto-redemption: `" + strings.Title(platform) + "`\n"
	} else {
		content = ":warning: No platform set for SHiFT code auto-redemption.\n"
	}
	content += "Do I currently DM you on successful code redemptions: `"
	if shouldDM {
		content += "Yes`"
	} else {
		content += "No`"
	}
	content += "\n\nFeel free to change your settings using the controls below:"
	msg := privateMessageResponse(content)
	msg.Data.Components = []discordgo.MessageComponent{
		DMComponents,
		PlatformComponents,
	}
	return msg
}
