package commands

import (
	"github.com/bwmarrin/discordgo"
)

type Command struct {
	ApplicationCommand *discordgo.ApplicationCommand
	Handler            func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var Commands = make(map[string]Command)

var VoiceStates = make(map[string]*VoiceState)

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

func UpdateBotStatus(s *discordgo.Session, status string, activityType discordgo.ActivityType, activityName string) error {
	activity := discordgo.Activity{
		Name: activityName,
		Type: activityType,
	}

	updateData := discordgo.UpdateStatusData{
		Activities: []*discordgo.Activity{&activity},
		Status:     status,
		AFK:        false,
	}

	return s.UpdateStatusComplex(updateData)
}
