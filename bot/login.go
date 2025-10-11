package bot

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/denverquane/slickshift/shift"
)

func (bot *Bot) loginCookieResponse(userID string, s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	if len(i.ApplicationCommandData().Options) == 0 {
		return cookieInstructionsResponse()
	}
	cookie := strings.TrimSpace(i.ApplicationCommandData().Options[0].StringValue())
	cookies := strings.Split(cookie, ";")
	newCookies := shift.ParseRequiredCookies(cookies)
	if len(newCookies) != 2 {
		return privateMessageResponse("Hm, doesn't look like you provided the right Cookie information...\n\n" +
			"Call `" + LOGIN + "` again without any values to see how to obtain the proper SHiFT cookies.")
	}
	client, err := shift.NewClient(newCookies)
	if err != nil {
		log.Println(err)
		return privateMessageResponse("I encountered an error creating an HTTP client for login. Please try again later.")
	}
	_, err = client.CheckRewards(shift.Steam, shift.Borderlands4, 0)
	if err != nil {
		log.Println(err)
		return privateMessageResponse("I encountered an error fetching the SHiFT rewards website with your Cookie. Are you sure you copy/pasted it correctly?")
	}
	err = bot.storage.EncryptAndSetUserCookies(userID, newCookies)
	if err != nil {
		log.Println(err)
		return privateMessageResponse("I logged into SHiFT with your info, but I wasn't able to store your session cookies for later...")
	}
	bot.triggerRedemptionProcessing(userID)
	return privateMessageResponse(Cheer + " Success! " + Cheer + "\n\nI've securely stored your session cookies for automatic SHiFT code redemption!")
}

const ImageURL = "https://i.imgur.com/3pNuoWM.png"

func cookieInstructionsResponse() *discordgo.InteractionResponse {
	content := "Wondering how to get your SHiFT cookies to login?\n" +
		"Use the image linked down below, and follow these instructions:\n\n" +
		"1. In your web browser, open the console (typically with the F12 key).\n" +
		"2. Open the \"Network\" tab.\n" +
		"3. Navigate to https://shift.gearboxsoftware.com/rewards and make sure you're logged in.\n" +
		"4. Find a request labelled \"rewards\" and click it.\n" +
		"5. Under \"Request Headers\" on the right, locate the \"Cookie\" field.\n" +
		"6. Copy the *entire* block of text to right of the \"Cookie\" label. It should contain `si=...` and also `_session_id=..`.\n" +
		"7. In Discord, call my `/" + LOGIN + "` command again using this value.\n\n" +
		"If you did everything right, I should be able to automatically redeem codes for you using these cookies!"
	msg := privateMessageResponse(content)
	msg.Data.Embeds = []*discordgo.MessageEmbed{
		{
			Image: &discordgo.MessageEmbedImage{
				URL: ImageURL,
			},
		},
	}
	return msg
}
