package bot

import "github.com/bwmarrin/discordgo"

func (bot *Bot) helpResponse(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	var embeds []*discordgo.MessageEmbed
	for _, command := range AllCommands {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:       "`/" + command.Name + "`",
			Description: "`" + command.Description + "`",
		})
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: PrivateResponse,
			Content: "SlickShift is a bot that can redeem Borderlands 4 SHiFT codes for you!\n\n" +
				"Below are the commands you can use to interact with SlickShift:",
			Embeds: embeds,
		},
	}
}
