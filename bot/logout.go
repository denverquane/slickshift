package bot

import (
	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) logoutResponse(userID string, s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	exists := bot.storage.UserCookiesExists(userID)
	if !exists {
		return privateMessageResponse("You are not logged in, so I have nothing to logout!")
	}
	msg := privateMessageResponse("Are you sure you want to logout? You'll have to login again if you want me to automatically redeem SHiFT codes for you again...")
	msg.Data.Components = []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label: "Yes, Logout",
					Style: discordgo.PrimaryButton,
					Emoji: &discordgo.ComponentEmoji{
						Name: ThumbsUp,
					},
					CustomID: LogoutPrefix + "true",
				},
				discordgo.Button{
					Label: "No, Don't Logout",
					Style: discordgo.DangerButton,
					Emoji: &discordgo.ComponentEmoji{
						Name: X,
					},
					CustomID: LogoutPrefix + "false",
				},
			},
		},
	}
	return msg
}
