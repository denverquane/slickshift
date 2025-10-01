package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/denverquane/slickshift/shift"
)

const (
	HELP        = "help"
	SETTINGS    = "settings"
	SECURITY    = "security"
	LOGIN       = "login"
	LOGINCOOKIE = "login-cookie"
	ADD         = "add"
	REDEMPTIONS = "redemptions"
)

var AllCommands = []*discordgo.ApplicationCommand{
	{
		Name:        HELP,
		Description: "View Help information and how to use SlickShift",
	},
	{
		Name:        SETTINGS,
		Description: "View and/or change the settings used for redeeming SHiFT codes",
	},
	{
		Name:        SECURITY,
		Description: "View information on how SlickShift securely handles your credentials and data",
	},
	{
		Name:        LOGIN,
		Description: "Login using your SHiFT credentials",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "email",
				Description: "SHiFT email address",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "password",
				Description: "SHiFT password",
				Required:    true,
			},
		},
	},
	{
		Name:        LOGINCOOKIE,
		Description: "Authenticate using a Cookie obtained manually from the SHiFT website",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "cookie",
				Description: "Cookie (containing si= and _session_id=) as provided by the SHiFT website",
				Required:    true,
			},
		},
	},
	{
		Name:        ADD,
		Description: "Add a new SHiFT code",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "code",
				Description: "SHiFT code",
				Required:    true,
				MinLength:   &shift.CodeLength,
				MaxLength:   shift.CodeLength,
			},
		},
	},
	{
		Name:        REDEMPTIONS,
		Description: "View recent redemptions SlickShift has performed for you",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "successful",
				Description: "Only retrieve successful redemptions",
				Required:    false,
			},
		},
	},
}

var PlatformComponents = discordgo.ActionsRow{
	Components: []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Steam",
			Style: discordgo.PrimaryButton,
			//Emoji: &discordgo.ComponentEmoji{
			//	Name: ,
			//},
			CustomID: SetPlatformPrefix + string(shift.Steam),
		},
		discordgo.Button{
			Label: "Epic",
			Style: discordgo.PrimaryButton,
			//Emoji: &discordgo.ComponentEmoji{
			//	Name: ,
			//},
			CustomID: SetPlatformPrefix + string(shift.Epic),
		},
		discordgo.Button{
			Label: "Xbox",
			Style: discordgo.PrimaryButton,
			//Emoji: &discordgo.ComponentEmoji{
			//	Name: ,
			//},
			CustomID: SetPlatformPrefix + string(shift.XboxLive),
		},
		discordgo.Button{
			Label: "Playstation",
			Style: discordgo.PrimaryButton,
			//Emoji: &discordgo.ComponentEmoji{
			//	Name: ,
			//},
			CustomID: SetPlatformPrefix + string(shift.PSN),
		},
	},
}

var one = 0

var DMComponents = discordgo.ActionsRow{
	Components: []discordgo.MessageComponent{
		discordgo.SelectMenu{
			CustomID:    SetDMPrefix,
			Placeholder: "Choose one...",
			MinValues:   &one,
			MaxValues:   1,
			Options: []discordgo.SelectMenuOption{
				{
					Label: "No, don't DM me",
					Value: "false",
					Emoji: &discordgo.ComponentEmoji{
						Name: "❌",
					},
				},
				{
					Label: "Yes, please DM me",
					Value: "true",
					Emoji: &discordgo.ComponentEmoji{
						Name: "✅",
					},
				},
			},
		},
	},
}
