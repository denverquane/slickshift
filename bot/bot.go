package bot

import (
	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	PrivateResponse   = discordgo.MessageFlagsEphemeral
	SetPlatformPrefix = "set_platform_"
	//GithubLink        = "https://github.com/denverquane/slickshift/releases"
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
			fallthrough
		case PLATFORM:
			return &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   PrivateResponse,
					Content: i.ApplicationCommandData().Name,
				},
			}
		case LOGIN:
			email := i.ApplicationCommandData().Options[0].StringValue()
			password := i.ApplicationCommandData().Options[1].StringValue()

			client, err := shift.NewClient()
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

			cookie, err := client.Login(email, password)
			if err != nil {
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
			err = bot.storage.SetUserCookie(i.Member.User.ID, cookie)
			if err != nil {
				log.Println(err)
				err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   PrivateResponse,
						Content: "I logged into SHiFT with your info, but I wasn't able to store your session cookie for later...",
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
					Content: "Success! I've securely stored your session cookie (and purged your email/password) for automatic SHiFT code redemption!",
				},
			}
		}
	} else if i.Type == discordgo.InteractionMessageComponent {
		id := i.MessageComponentData().CustomID
		if strings.HasPrefix(id, SetPlatformPrefix) {
			id = strings.TrimPrefix(id, SetPlatformPrefix)
			var setPlat string
			switch id {
			case "steam":
				setPlat = "steam"
			case "xbox":
				setPlat = "xbox"
			case "playstation":
				setPlat = "playstation"
			}
			if setPlat != "" {
				err := bot.storage.AddUser(i.Member.User.ID, setPlat)
				if err != nil {
					log.Println(err)
					return nil
				}
				return &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags: PrivateResponse,
						Content: "Success!\nThe next step is to setup authentication with SHiFT so that I " +
							"can redeem codes for you automatically in the future!\n\n" +
							"At present, the only method for login is by providing your email/password directly with `/login`",
					},
				}
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
				"I'm here to help you automatically redeem SHiFT codes for Borderlands 4!\n\n" +
				"To start, can you tell me what platform you'll want to redeem SHiFT codes for?",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label: "Steam",
							Style: discordgo.PrimaryButton,
							//Emoji: &discordgo.ComponentEmoji{
							//	Name: ,
							//},
							CustomID: SetPlatformPrefix + "steam",
						},
						discordgo.Button{
							Label: "Xbox",
							Style: discordgo.PrimaryButton,
							//Emoji: &discordgo.ComponentEmoji{
							//	Name: ,
							//},
							CustomID: SetPlatformPrefix + "xbox",
						},
						discordgo.Button{
							Label: "Playstation",
							Style: discordgo.PrimaryButton,
							//Emoji: &discordgo.ComponentEmoji{
							//	Name: ,
							//},
							CustomID: SetPlatformPrefix + "ps",
						},
					},
				},
			},
		},
	}
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
