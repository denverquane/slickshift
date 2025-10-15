package bot

import (
	"log"
	"log/slog"
	"strings"

	"github.com/denverquane/slickshift/store"

	"github.com/bwmarrin/discordgo"
)

const (
	PrivateResponse   = discordgo.MessageFlagsEphemeral
	SetPlatformPrefix = "set_platform_"
	SetDMPrefix       = "set_dm_value"
	LogoutPrefix      = "logout_"
	GithubLink        = "https://github.com/denverquane/slickshift"
	SecurityLink      = GithubLink + "/blob/main/SECURITY.md"
	LiabilityLink     = GithubLink + "/blob/main/LIABILITY.md"
	ServerLink        = "https://discord.gg/GDSsKcrPxp"
	BotInviteLink     = "https://discord.com/oauth2/authorize?client_id=1420238749270544547"
	ThumbsUp          = "👍"
	X                 = "❌"
	Lock              = "🔒"
	Cheer             = "🎉"
)

type Bot struct {
	session           *discordgo.Session
	storage           store.Store
	redemptionTrigger chan string
	version           string
	commit            string
}

func CreateNewBot(token string, storage store.Store, version, commit string) (*Bot, error) {
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		session:           discord,
		storage:           storage,
		redemptionTrigger: make(chan string, 10),
		version:           version,
		commit:            commit,
	}, nil
}

func (bot *Bot) Start() error {
	bot.session.AddHandler(bot.handleSlashCommand)

	bot.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is now online according to discord Ready handler")
	})
	return bot.session.Open()
}

func (bot *Bot) handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID string
	if i.Member == nil || i.Member.User == nil {
		if i.User != nil {
			userID = i.User.ID
		} else {
			log.Println("Could not ascertain userID from i.Member.User nor i.User")
			return
		}
	} else {
		userID = i.Member.User.ID
	}
	if userID == "" {
		log.Println("User ID is empty")
		return
	}

	exists := bot.storage.UserExists(userID)
	if i.Type == discordgo.InteractionApplicationCommand && !exists {
		err := s.InteractionRespond(i.Interaction, unregisteredUserResponse())
		if err != nil {
			log.Println(err)
		}
		return
	}

	resp := bot.getSlashResponse(userID, s, i)
	if resp != nil {
		err := s.InteractionRespond(i.Interaction, resp)
		if err != nil {
			log.Println(err)
		}
	}
}

func (bot *Bot) getSlashResponse(userID string, s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case HELP:
			return bot.helpResponse(s, i)
		case SETTINGS:
			return bot.settingsResponse(userID, s, i)
		case LOGIN_INSECURE:
			return bot.loginResponse(userID, s, i)
		case LOGIN:
			return bot.loginCookieResponse(userID, s, i)
		case LOGOUT:
			return bot.logoutResponse(userID, s, i)
		case ADD:
			return bot.addResponse(userID, s, i)
		case INFO:
			return bot.infoResponse(userID, s, i)
		case REDEMPTIONS:
			return bot.redemptionsResponse(userID, s, i)
		}
	} else if i.Type == discordgo.InteractionMessageComponent {
		exists := bot.storage.UserExists(userID)
		if !exists {
			err := bot.storage.AddUser(userID)
			if err != nil {
				log.Println(err)
				return nil
			}
		}
		oldPlatform, _, err := bot.storage.GetUserPlatformAndDM(userID)
		if err != nil {
			log.Println(err)
			return nil
		}
		id := i.MessageComponentData().CustomID
		if strings.HasPrefix(id, SetPlatformPrefix) {
			platform := strings.TrimPrefix(id, SetPlatformPrefix)

			err := bot.storage.SetUserPlatform(userID, platform)
			if err != nil {
				log.Println(err)
				return nil
			}
			// if they set the platform for the first time, or are a new user, then send a different response
			if !exists || oldPlatform == "" {
				return registeredUserResponse()
			}
			return privateMessageResponse("Got it!\nSet the platform for future redemptions to: `" + strings.Title(platform) + "`")

		} else if strings.HasPrefix(id, SetDMPrefix) {
			if len(i.MessageComponentData().Values) == 0 {
				return privateMessageResponse("Hm, I couldn't process that. If you're trying to set the DM preference, try with `/" + SETTINGS + "`")
			}
			dm := i.MessageComponentData().Values[0] == "true"
			err := bot.storage.SetUserDM(userID, dm)
			if err != nil {
				return privateMessageResponse("Hm, I got an error trying to set your DM preference. Please try again later.")
			}
			if dm {
				go func() {
					err = bot.DMUser(userID, "Hey there!\nWas just confirming I could send you a message.\nThanks!")
					if err != nil {
						err = s.InteractionRespond(i.Interaction, privateMessageResponse(
							"Hm, doesn't look like I was able to send you a Direct Message...\n"+
								"Do you have the Discord Setting \"Allow Direct Messages from Server Members\" enabled?\n"+
								"You'll need it enabled for whatever server(s) you and I are both members of."),
						)
						if err != nil {
							log.Println(err)
						}
					} else {
						s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseDeferredMessageUpdate,
						})
					}
				}()
			} else {
				return &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseDeferredMessageUpdate,
				}
			}
		} else if strings.HasPrefix(id, LogoutPrefix) {
			value := strings.TrimPrefix(id, LogoutPrefix)
			if value == "true" {
				err = bot.storage.DeleteUserCookies(userID)
				if err != nil {
					log.Println(err)
					return privateMessageResponse("Hm, I had an issue deleting your cookies. Please try again later.")
				}
				msg := privateMessageResponse(ThumbsUp + " You've been successfully logged out!")
				msg.Data.Components = []discordgo.MessageComponent{}
				return msg
			}
			msg := privateMessageResponse(ThumbsUp)
			msg.Data.Components = []discordgo.MessageComponent{}
			return msg
		}
	}

	return nil
}

func privateMessageResponse(content string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   PrivateResponse,
			Content: content,
		},
	}
}

func unregisteredUserResponse() *discordgo.InteractionResponse {
	msg := privateMessageResponse("Looks like this is your first time using SlickShift! Welcome!\n\n" +
		"I'm here to help you automatically redeem SHiFT codes for Borderlands 4!\n\n*To get started:*\n" +
		settingsSuffix)
	msg.Data.Components = []discordgo.MessageComponent{
		getDMComponents(false, false),
		getPlatformComponents(false, ""),
	}
	return msg
}

func registeredUserResponse() *discordgo.InteractionResponse {
	content := Cheer + "  Success!  " + Cheer + "\n\n" +
		"The next step is to setup authentication with SHiFT so that I can redeem codes for you automatically in the future:\n" +
		"* (Recommended) Provide a Cookie you obtain yourself from the SHiFT website with `/" + LOGIN + "`\n" +
		"* Provide your SHiFT email/password directly with `/" + LOGIN_INSECURE + "`\n\n" +
		"To see more details about the differences between these two options, see [SECURITY.md](" + SecurityLink + ")\n" +
		"** By logging in and continuing to use SlickShift, you agree to the [Liability Disclosure](" + LiabilityLink + ")**"
	return privateMessageResponse(content)
}

func (bot *Bot) DMUser(userID, content string) error {
	channel, err := bot.session.UserChannelCreate(userID)
	if err != nil {
		return err
	}
	_, err = bot.session.ChannelMessageSend(channel.ID, content)
	return err
}

func (bot *Bot) RegisterCommands(guildID string) ([]*discordgo.ApplicationCommand, error) {
	cmds := make([]*discordgo.ApplicationCommand, len(AllCommands))
	for i, v := range AllCommands {
		cmd, err := bot.session.ApplicationCommandCreate(bot.session.State.User.ID, guildID, v)
		if err != nil {
			return nil, err
		}
		slog.Info("Registered command", "command", v.Name)
		cmds[i] = cmd
	}
	return cmds, nil
}

func (bot *Bot) DeleteCommands(guildID string, cmds []*discordgo.ApplicationCommand) {
	for _, v := range cmds {
		err := bot.session.ApplicationCommandDelete(v.ApplicationID, guildID, v.ID)
		if err != nil {
			log.Printf("Error deleting %v: %v", v.Name, err)
		}
	}
}

// trigger redemption processing for whatever userID was provided. If empty, triggers for all users
func (bot *Bot) triggerRedemptionProcessing(userID string) {
	bot.redemptionTrigger <- userID
}

func (bot *Bot) Stop() error {
	err := bot.storage.Close()
	if err != nil {
		slog.Error("Error closing storage", "error", err)
	}
	return bot.session.Close()
}
