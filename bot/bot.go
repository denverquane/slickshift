package bot

import (
	"log"
	"log/slog"
	"strings"

	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"

	"github.com/bwmarrin/discordgo"
)

const (
	PrivateResponse   = discordgo.MessageFlagsEphemeral
	SetPlatformPrefix = "set_platform_"
	SetDMPrefix       = "set_dm_value"
	LogoutPrefix      = "logout_"
	GithubLink        = "https://github.com/denverquane/slickshift"
	ThumbsUp          = "👍"
	X                 = "❌"
	Lock              = "🔒"
	Cheer             = "🎉"
)

const (
	Red    = 15548997
	Green  = 5763719
	Yellow = 16705372
)

type Bot struct {
	session *discordgo.Session
	storage store.Store
}

func CreateNewBot(token string, storage store.Store) (*Bot, error) {
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		session: discord,
		storage: storage,
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
		case SECURITY:
			return bot.securityResponse(s, i)
		case SETTINGS:
			return bot.settingsResponse(userID, s, i)
		case LOGIN_INSECURE:
			return bot.loginResponse(userID, s, i)
		case LOGIN:
			return bot.loginCookieResponse(userID, s, i)
		case LOGOUT:
			return bot.logoutResponse(userID, s, i)
		case ADD:
			code := i.ApplicationCommandData().Options[0].StringValue()
			if !shift.CodeRegex.MatchString(code) {
				return privateMessageResponse("Hm, doesn't look like you provided a valid SHiFT code. It should look something like:\n\n" +
					"`XXXX-XXXXX-XXXXX-XXXXX-XXXXX`")

			}
			if bot.storage.CodeExists(code) {
				return privateMessageResponse("It looks like that code already exists!\nThanks anyways!")
			}
			var src = store.DiscordSource
			err := bot.storage.AddCode(code, string(shift.Borderlands4), &userID, &src)
			if err != nil {
				log.Println(err)
				return nil
			}
			return privateMessageResponse("Nice, thanks for adding the code! It should be tested and validated soon!")
		case STATS:
			return bot.statsResponse(s, i)
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

			// TODO verify the id here
			err := bot.storage.SetUserPlatform(userID, platform)
			if err != nil {
				log.Println(err)
				return nil
			}
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
						err = s.InteractionRespond(i.Interaction, privateMessageResponse("Hm, doesn't look like I was able to send you a Direct Message... "+
							"Are you sure you haven’t disabled DMs from server members?"),
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
		DMComponents,
		PlatformComponents,
	}
	return msg
}

func registeredUserResponse() *discordgo.InteractionResponse {
	content := "Success!\nThe next step is to setup authentication with SHiFT so that I " +
		"can redeem codes for you automatically in the future!\n\n" +
		"There are two different options available at this time:\n" +
		"* (Recommended) Provide a Cookie you obtain yourself from the SHiFT website with `/" + LOGIN + "`\n" +
		"* Provide your SHiFT email/password directly with `/" + LOGIN_INSECURE + "`\n" +
		"To see more details about the differences between these two options, try `/" + SECURITY + "`"
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

func (bot *Bot) Stop() error {
	return bot.session.Close()
}
