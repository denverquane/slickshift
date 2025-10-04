package bot

import "github.com/bwmarrin/discordgo"

func (bot *Bot) securityResponse(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: Lock + " Security Overview " + Lock + "\n\n" +
				"SlickShift does not store **any** usernames or passwords. \n" +
				"Instead, it uses SHiFT website session cookies, which are encrypted in the Bot's database. " +
				"These cookies are only used to log you into SHiFT Rewards so the bot can redeem codes for you — and nothing else.\n\n" +
				"Your data is never shared with third parties, and the entire project is completely open-source, so anyone can review exactly how it works [on GitHub](" + GithubLink + ")\n\n" +
				"`/" + LOGIN_INSECURE + "` vs. `/" + LOGIN + "`\n\n" +
				"If you specify your login details with `/" + LOGIN_INSECURE + "`, SlickShift uses your username/password to login to SHiFT on your behalf, obtain session cookies, and then encrypt and store those cookies for later. " +
				"Once this process completes, *your username/password are completely forgotten/discarded by the bot.*\n\n" +
				"`/" + LOGIN + "` takes a different approach, and instead requires users to provide session cookies directly; no username or password. This is more secure, but is " +
				"more cumbersome, as it requires users to acquire these cookies themselves and then input them.\n\n" +
				"To view steps on how to obtain your cookie and login securely, simply call `/" + LOGIN + "` with no arguments.",
		},
	}
}
