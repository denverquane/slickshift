package bot

import (
	"fmt"
	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"
	"sort"

	"github.com/bwmarrin/discordgo"
)

func percentFormatted(num, denom int64) string {
	if num == 0 || num == denom {
		return ""
	}
	v := 100.0 * float64(num) / float64(denom)
	return fmt.Sprintf(" (%.0f%%)", v)
}

func sortedUserEmbeds(stats store.Statistics) []*discordgo.MessageEmbedField {
	type kv struct {
		Key   shift.Platform
		Value int64
	}

	var ss []kv
	for k, v := range stats.Users {
		if k != shift.Total {
			ss = append(ss, kv{k, v})
		}
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	total := stats.Users[shift.Total]
	embeds := make([]*discordgo.MessageEmbedField, len(ss)+1)
	embeds[0] = &discordgo.MessageEmbedField{
		Name:   "Total",
		Value:  fmt.Sprintf("%d", total),
		Inline: false,
	}
	for i, elem := range ss {
		embeds[i+1] = &discordgo.MessageEmbedField{
			Name:   shift.ToPretty(elem.Key),
			Value:  fmt.Sprintf("%d%s", elem.Value, percentFormatted(elem.Value, total)),
			Inline: true,
		}
	}
	return embeds
}

func (bot *Bot) infoResponse(userID string, s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	stats, err := bot.storage.GetStatistics(userID)
	if err != nil {
		return privateMessageResponse("Hm, I got an error fetching statistics. Please try again later.")
	}
	embeds := []*discordgo.MessageEmbed{
		&discordgo.MessageEmbed{
			Title:  "Users",
			Fields: sortedUserEmbeds(stats),
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
					Name:   "Total",
					Value:  fmt.Sprintf("%d", stats.Redemptions),
					Inline: false,
				},
				&discordgo.MessageEmbedField{
					Name:   "Successful",
					Value:  fmt.Sprintf("%d %s", stats.SuccessRedemptions, percentFormatted(stats.SuccessRedemptions, stats.Redemptions)),
					Inline: true,
				},
			},
			Color: Green,
		},
		&discordgo.MessageEmbed{
			Title: "Project",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:  "Repository",
					Value: GithubLink,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: bot.version + "-" + bot.commit,
			},
		},
	}
	msg := privateMessageResponse("SlickShift Statistics")
	msg.Data.Embeds = embeds
	return msg
	//Timestamp:   time.Unix(red.TimeUnix, 0).UTC().Format(time.RFC3339),
}
