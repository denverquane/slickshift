package bot

import (
	"github.com/bwmarrin/discordgo"
)

const (
	HELP     = "help"
	PLATFORM = "platform"
	LOGIN    = "login"
)

var AllCommands = []*discordgo.ApplicationCommand{
	{
		Name:        HELP,
		Description: "View Help information and how to use SlickShift",
	},
	{
		Name:        PLATFORM,
		Description: "View or change the platform for redeeming SHiFT codes",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "platform",
				Description: "Platform for redeeming SHiFT codes",
				Required:    false,
				Options:     []*discordgo.ApplicationCommandOption{},
			},
		},
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
}
