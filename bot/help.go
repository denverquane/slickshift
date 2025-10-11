package bot

import "github.com/bwmarrin/discordgo"

func (bot *Bot) helpResponse(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	return privateMessageResponse("SlickShift is a bot that can redeem Borderlands 4 SHiFT codes for you!\n\n" +
		"The first recommended step is to call `/" + LOGIN + "` with no arguments to see steps on how to securely login.\n" +
		"If you've read the information provided by [SECURITY.md](" + SecurityLink + ") and **understand the implications**, you can alternatively use `/" + LOGIN_INSECURE + "`\n\n" +
		"If you're looking to get support, request new features, or just chat about the Bot, feel free to join the Discord here!\n" + ServerLink)
}
