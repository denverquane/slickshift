package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/denverquane/slickshift/shift"
	"log"
)

func (bot *Bot) loginResponse(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	email := i.ApplicationCommandData().Options[0].StringValue()
	password := i.ApplicationCommandData().Options[1].StringValue()

	client, err := shift.NewClient(nil)
	if err != nil {
		log.Println(err)
		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   PrivateResponse,
				Content: "I encountered an error creating an HTTP client for login. Please try again later.",
			},
		}
	}

	err = client.Login(email, password)
	if err != nil {
		log.Println(err)
		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   PrivateResponse,
				Content: "I wasn't able to log you in to SHiFT. Are you sure you provided the right credentials?",
			},
		}
	}
	cookies := client.DumpCookies()
	err = bot.storage.EncryptAndSetUserCookies(i.Member.User.ID, cookies)
	if err != nil {
		log.Println(err)
		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   PrivateResponse,
				Content: "I logged into SHiFT with your info, but I wasn't able to store your session cookies for later...",
			},
		}
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   PrivateResponse,
			Content: "Success! :tada:\nI've securely stored your session cookies (and purged your email/password) for automatic SHiFT code redemption!",
		},
	}
}
