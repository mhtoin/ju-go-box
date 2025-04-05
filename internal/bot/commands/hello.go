package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func init() {
	RegisterCommand(Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "hello",
			Description: "Get a friendly greeting from the bot",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Hello, %s!", i.Member.User.Username),
				},
			})
		},
	})
} 