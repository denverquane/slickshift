package bot

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"strings"
)

const settingsSuffix = "* Do you want me to message you when I redeem codes for you, or when your login details expire?\n" +
	"* Also, can you tell me what platform you'd like to auto-redeem SHiFT codes for?\n"

func (bot *Bot) settingsResponse(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	userID := i.Member.User.ID
	// TODO consolidate
	platform, err := bot.storage.GetUserPlatform(userID)
	if err != nil {
		log.Println(err)
		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   PrivateResponse,
				Content: "Hm, I got an error fetching your platform. Please try again later.",
			},
		}
	}
	shouldDM, err := bot.storage.GetUserDM(userID)
	if err != nil {
		log.Println(err)
		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   PrivateResponse,
				Content: "Hm, I got an error fetching your dm setting. Please try again later.",
			},
		}
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
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: PrivateResponse,
			Components: []discordgo.MessageComponent{
				DMComponents,
				PlatformComponents,
			},
			Content: content,
		},
	}
}
