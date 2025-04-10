package commands

import (
	"github.com/bwmarrin/discordgo"
)

func init() {
	RegisterCommand(Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "stop",
			Description: "Stop music playback",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			voiceConnection, ok := s.VoiceConnections[i.GuildID]
			if !ok {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "I am not connected to a voice channel",
					},
				})
				return
			}

			if voiceState, ok := VoiceStates[i.GuildID]; ok && voiceState != nil {
				select {
				case voiceState.StopChannel <- true:
				default:
				}
				delete(VoiceStates, i.GuildID)
			}

			voiceConnection.Close()

			UpdateBotStatus(s, "online", discordgo.ActivityTypeCustom, "Ready to play music")

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Stopped playback and disconnected from voice channel",
				},
			})
		},
	})
}
