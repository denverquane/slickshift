package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (bot *Bot) statsResponse(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	stats, err := bot.storage.GetStatistics()
	if err != nil {
		return privateMessageResponse("Hm, I got an error fetching statistics. Please try again later.")
	}
	embeds := []*discordgo.MessageEmbed{
		&discordgo.MessageEmbed{
			Title: "Users",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "Total",
					Value:  fmt.Sprintf("%d", stats.Users),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Steam",
					Value:  fmt.Sprintf("%d", stats.SteamUsers),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Epic",
					Value:  fmt.Sprintf("%d", stats.EpicUsers),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Xbox",
					Value:  fmt.Sprintf("%d", stats.XboxUsers),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Playstation",
					Value:  fmt.Sprintf("%d", stats.PsnUsers),
					Inline: true,
				},
			},
		},
		&discordgo.MessageEmbed{
			Title: "Codes",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "Total",
					Value:  fmt.Sprintf("%d", stats.Codes),
					Inline: true,
				},
			},
			Color: Yellow,
		},
		&discordgo.MessageEmbed{
			Title: "Redemptions",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "Successful",
					Value:  fmt.Sprintf("%d", stats.SuccessRedemptions),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Total",
					Value:  fmt.Sprintf("%d", stats.Redemptions),
					Inline: true,
				},
			},
			Color: Green,
		},
	}
	msg := privateMessageResponse("SlickShift Statistics")
	msg.Data.Embeds = embeds
	return msg
	//Timestamp:   time.Unix(red.TimeUnix, 0).UTC().Format(time.RFC3339),
}
