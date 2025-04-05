package commands

import (
	"github.com/bwmarrin/discordgo"
)

type Command struct {
	ApplicationCommand *discordgo.ApplicationCommand
	Handler           func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var Commands = make(map[string]Command)

func RegisterCommand(cmd Command) {
	Commands[cmd.ApplicationCommand.Name] = cmd
}

func GetApplicationCommands() []*discordgo.ApplicationCommand {
	var appCommands []*discordgo.ApplicationCommand
	for _, cmd := range Commands {
		appCommands = append(appCommands, cmd.ApplicationCommand)
	}
	return appCommands
} 