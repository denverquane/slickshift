package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/denverquane/slickshift/shift"
	"log"
	"strings"
)

func (bot *Bot) loginCookieResponse(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	cookie := strings.TrimSpace(i.ApplicationCommandData().Options[0].StringValue())
	cookies := strings.Split(cookie, ";")
	newCookies := shift.ParseRequiredCookies(cookies)
	if len(newCookies) != 2 {
		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: PrivateResponse,
				Content: "Hm, doesn't look like you provided the right Cookie... It should look something like:\n\n`" +
					"si=lots_of_text_here; _session_id=more_text_here`",
			},
		}
	}
	client, err := shift.NewClient(newCookies)
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
	_, err = client.CheckRewards(shift.Steam, shift.Borderlands4, 0)
	if err != nil {
		log.Println(err)
		return &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   PrivateResponse,
				Content: "I encountered an error fetching the SHiFT rewards website with your Cookie. Are you sure you copy/pasted it correctly?",
			},
		}
	}
	err = bot.storage.EncryptAndSetUserCookies(i.Member.User.ID, newCookies)
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
			Content: "Success! :tada:\nI've securely stored your session cookies for automatic SHiFT code redemption!",
		},
	}
}
