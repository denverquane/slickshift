package bot

import (
	"log"
	"strings"
	"time"

	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"

	"github.com/bwmarrin/discordgo"
)

const (
	PrivateResponse   = discordgo.MessageFlagsEphemeral
	SetPlatformPrefix = "set_platform_"
	SetDMPrefix       = "set_dm_value"
	//GithubLink        = "https://github.com/denverquane/slickshift/releases"
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
	userID := i.Member.User.ID
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

	resp := bot.getSlashResponse(s, i)
	if resp != nil {
		err := s.InteractionRespond(i.Interaction, resp)
		if err != nil {
			log.Println(err)
		}
	}

}

func (bot *Bot) getSlashResponse(s *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case HELP:
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
		case SECURITY:
			// TODO
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   PrivateResponse,
					Content: "Super secure, bro. Trust.",
				},
			}
		case PLATFORM:
			platform, err := bot.storage.GetUserPlatform(i.Member.User.ID)
			if err != nil {
				log.Println(err)
				return &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   PrivateResponse,
						Content: "Hm, I got an error fetching your platform. Please try again later.",
					},
				}
			}
			var content string
			if platform == "" {
				content = "Please set your platform using the Buttons below!"
			} else {
				content = "Your current platform for SHiFT code auto-redemption is `" + strings.Title(platform) + "`\n" +
					"If you wish to change it, please use the Buttons below!"
			}
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: PrivateResponse,
					Components: []discordgo.MessageComponent{
						PlatformComponents,
					},
					Content: content,
				},
			}
		case LOGIN:
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
				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   PrivateResponse,
						Content: "I wasn't able to log you in to SHiFT. Are you sure you provided the right credentials?",
					},
				})
				if err != nil {
					log.Println(err)
				}
				return nil
			}
			cookies := client.DumpCookies()
			err = bot.storage.EncryptAndSetUserCookies(i.Member.User.ID, cookies)
			if err != nil {
				log.Println(err)
				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   PrivateResponse,
						Content: "I logged into SHiFT with your info, but I wasn't able to store your session cookies for later...",
					},
				})
				if err != nil {
					log.Println(err)
				}
				return nil
			}
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   PrivateResponse,
					Content: "Success! I've securely stored your session cookies (and purged your email/password) for automatic SHiFT code redemption!",
				},
			}
		case LOGINCOOKIE:
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
				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   PrivateResponse,
						Content: "I logged into SHiFT with your info, but I wasn't able to store your session cookies for later...",
					},
				})
				if err != nil {
					log.Println(err)
				}
				return nil
			}
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   PrivateResponse,
					Content: "Success! I've securely stored your session cookies for automatic SHiFT code redemption!",
				},
			}
		case ADD:
			code := i.ApplicationCommandData().Options[0].StringValue()
			if !shift.CodeRegex.MatchString(code) {
				return &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags: PrivateResponse,
						Content: "Hm, doesn't look like you provided a valid SHiFT code. It should look something like:\n\n" +
							"`XXXX-XXXXX-XXXXX-XXXXX-XXXXX`",
					},
				}
			}
			if bot.storage.CodeExists(code) {
				return &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   PrivateResponse,
						Content: "It looks like that code already exists!\nThanks anyways!",
					},
				}
			}
			var src = store.DiscordSource
			err := bot.storage.AddCode(code, string(shift.Borderlands4), &i.Member.User.ID, &src)
			if err != nil {
				log.Println(err)
				return nil
			}
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   PrivateResponse,
					Content: "Nice, thanks for adding the code! It should be tested and validated soon!",
				},
			}
		case REDEMPTIONS:
			values := i.ApplicationCommandData().Options
			var status string
			if len(values) > 0 && values[0].BoolValue() {
				status = shift.SUCCESS
			}
			redemptions, err := bot.storage.GetRecentRedemptionsForUser(i.Member.User.ID, status, 3)
			if err != nil {
				log.Println(err)
				return &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   PrivateResponse,
						Content: "Yikes, I got an error fetching your redemptions. Please try again later.",
					},
				}
			}
			if len(redemptions) == 0 {
				return &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   PrivateResponse,
						Content: "Sorry, I don't have any of those redemptions for you yet!",
					},
				}
			}
			var embeds []*discordgo.MessageEmbed
			for _, red := range redemptions {
				var reward string
				var color int
				if red.Reward.Valid {
					reward = red.Reward.String
				} else {
					reward = "*Reward Unknown*"
				}
				switch red.Status {
				case shift.SUCCESS:
					color = Green
				case shift.ALREADY_REDEEMED:
					color = Yellow
				case shift.NOT_EXIST:
				case shift.EXPIRED:
					color = Red
				}
				embeds = append(embeds, &discordgo.MessageEmbed{
					Title: reward,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Code",
							Value: red.Code,
						},
						//&discordgo.MessageEmbedField{
						//	Name:  "Game",
						//	Value: red.Game,
						//},
					},
					Color:       color,
					Description: red.Status,
					Timestamp:   time.Unix(red.TimeUnix, 0).UTC().Format(time.RFC3339),
				})
			}
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:  PrivateResponse,
					Embeds: embeds,
				},
			}
		}
	} else if i.Type == discordgo.InteractionMessageComponent {
		exists := bot.storage.UserExists(i.Member.User.ID)
		if !exists {
			err := bot.storage.AddUser(i.Member.User.ID)
			if err != nil {
				log.Println(err)
				return nil
			}
		}
		oldPlatform, err := bot.storage.GetUserPlatform(i.Member.User.ID)
		if err != nil {
			log.Println(err)
			return nil
		}
		id := i.MessageComponentData().CustomID
		if strings.HasPrefix(id, SetPlatformPrefix) {
			platform := strings.TrimPrefix(id, SetPlatformPrefix)

			// TODO verify the id here
			err := bot.storage.SetUserPlatform(i.Member.User.ID, platform)
			if err != nil {
				log.Println(err)
				return nil
			}
			if !exists || oldPlatform == "" {
				return registeredUserResponse()
			}
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   PrivateResponse,
					Content: "Got it!\nSet the platform for future redemptions to: `" + strings.Title(platform) + "`",
				},
			}
		} else if strings.HasPrefix(id, SetDMPrefix) {
			value := i.MessageComponentData().Values[0]
			dm := value == "true"
			err := bot.storage.SetUserDM(i.Member.User.ID, dm)
			if err != nil {
				return &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   PrivateResponse,
						Content: "Hm, I got an error trying to set your DM preference. Please try again later.",
					},
				}
			}
			if dm {
				go func() {
					err = bot.DMUser(i.Member.User.ID, "Hey there!\nWas just confirming I could send you a message.\nThanks!")
					if err != nil {
						err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Flags: PrivateResponse,
								Content: "Hm, doesn't look like I was able to send you a Direct Message... Are you sure you " +
									"haven’t disabled DMs from server members?",
							},
						})
						if err != nil {
							log.Println(err)
						}
					}
				}()
			}
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredMessageUpdate,
			}
		}
	}

	return nil
}

func unregisteredUserResponse() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: PrivateResponse,
			Content: "Looks like this is your first time using SlickShift! Welcome!\n\n" +
				"I'm here to help you automatically redeem SHiFT codes for Borderlands 4!\n\n*To get started:*\n" +
				"* Do you want me to message you when I redeem codes for you, or when your login details expire?\n" +
				"* Also, can you tell me what platform you'd like to auto-redeem SHiFT codes for?\n",
			Components: []discordgo.MessageComponent{
				DMComponents,
				PlatformComponents,
			},
		},
	}
}

func registeredUserResponse() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: PrivateResponse,
			Content: "Success!\nThe next step is to setup authentication with SHiFT so that I " +
				"can redeem codes for you automatically in the future!\n\n" +
				"There are two different options available at this time:\n" +
				"* Provide your SHiFT email/password directly with `/" + LOGIN + "`\n" +
				"* Provide a Cookie you obtain yourself from the SHiFT website with `/" + LOGINCOOKIE + "`\n\n" +
				"To see more details about the differences between these two options, try `/" + SECURITY + "`",
		},
	}
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
