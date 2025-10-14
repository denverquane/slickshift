package bot

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func percentFormatted(num, denom int64) string {
	if num == 0 || num == denom {
		return ""
	}
	v := 100.0 * float64(num) / float64(denom)
	return fmt.Sprintf(" (%.0f%%)", v)
}

type kv struct {
	Key   string
	Value int64
}

func sortMap(m map[string]int64) []kv {
	var ss []kv
	for k, v := range m {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})
	return ss
}

func toSortedEmbeds(m map[string]int64) []*discordgo.MessageEmbedField {
	sortedUsers := sortMap(m)

	embeds := make([]*discordgo.MessageEmbedField, len(sortedUsers))

	total := m["total"]
	embeds[0] = &discordgo.MessageEmbedField{
		Name:   "Total",
		Value:  fmt.Sprintf("%d", total),
		Inline: false,
	}

	i := 1
	for _, elem := range sortedUsers {
		if elem.Key != "total" {
			name := strings.Title(strings.ReplaceAll(elem.Key, "_", " "))
			embeds[i] = &discordgo.MessageEmbedField{
				Name:   name,
				Value:  fmt.Sprintf("%d%s", elem.Value, percentFormatted(elem.Value, total)),
				Inline: true,
			}
			i++
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
			Fields: toSortedEmbeds(stats.Users),
		},
		&discordgo.MessageEmbed{
			Title:  "Codes",
			Fields: toSortedEmbeds(stats.Codes),
			Color:  Yellow,
		},
		&discordgo.MessageEmbed{
			Title:  "Redemptions",
			Fields: toSortedEmbeds(stats.Redemptions),
			Color:  Green,
		},
		&discordgo.MessageEmbed{
			Title: "Slickshift",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:  "Official Server",
					Value: "[Join Server](" + ServerLink + ")",
				},
				&discordgo.MessageEmbedField{
					Name:  "Repository",
					Value: GithubLink,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "v" + bot.version + "-" + bot.commit,
			},
		},
	}
	msg := privateMessageResponse("SlickShift Statistics")
	msg.Data.Embeds = embeds
	return msg
	//Timestamp:   time.Unix(red.TimeUnix, 0).UTC().Format(time.RFC3339),
}
