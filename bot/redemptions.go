package bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/denverquane/slickshift/shift"
	"log/slog"
	"time"
)

func (bot *Bot) redemptionsResponse(userID string, session *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	var quantity = 3
	if len(i.ApplicationCommandData().Options) > 0 {
		quantity = int(i.ApplicationCommandData().Options[0].IntValue())
	}
	redemptions, err := bot.storage.GetRecentRedemptionsForUser(userID, "", quantity)
	if err != nil {
		slog.Error("Error fetching recent redemptions", "user_id", userID, "error", err.Error())
		return privateMessageResponse("Yikes, I got an error fetching your redemptions. Please try again later.")
	}
	if len(redemptions) == 0 {
		return privateMessageResponse("Looks like I don't have any redemptions for you yet!")
	}
	summary, err := bot.storage.RedemptionSummaryForUser(userID)
	if err != nil {
		slog.Error("Error fetching redemption summary", "user_id", userID, "error", err.Error())
		return privateMessageResponse("Yikes, I got an error fetching your redemption summary. Please try again later.")
	}
	var embeds = []*discordgo.MessageEmbed{
		{
			Title: "Redemption Summary",
			Color: Blue,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Total",
					Value:  fmt.Sprintf("%d", summary["total"]),
					Inline: true,
				},
				{
					Name:   "Successful",
					Value:  fmt.Sprintf("%d", summary["success"]),
					Inline: true,
				},
				{
					Name:   "Already Redeemed",
					Value:  fmt.Sprintf("%d", summary["already_redeemed"]),
					Inline: true,
				},
			},
		},
	}
	for _, redem := range redemptions {
		var reward string
		var color int
		if redem.Reward.Valid {
			reward = redem.Reward.String
		} else {
			reward = "*Reward Unknown*"
		}
		switch redem.Status {
		case shift.SUCCESS:
			color = Green
		case shift.ALREADY_REDEEMED:
			color = DarkOrange
		case shift.LINK2K:
			color = Yellow
		case shift.NOT_EXIST:
		case shift.EXPIRED:
			color = Red
		}
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title: reward,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Code",
					Value: redem.Code,
				},
				//&discordgo.MessageEmbedField{
				//	Name:  "Game",
				//	Value: red.Game,
				//},
			},
			Color:       color,
			Description: redem.Status,
			Timestamp:   time.Unix(redem.TimeUnix, 0).UTC().Format(time.RFC3339),
		})
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:  PrivateResponse,
			Embeds: embeds,
		},
	}
}
