package commands

import (
	"github.com/bwmarrin/discordgo"
)

func init() {
	RegisterCommand(Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "pause",
			Description: "Pause current audio playback",
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			_, ok := s.VoiceConnections[i.GuildID]
			if !ok {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "I am not connected to a voice channel",
					},
				})
				return
			}

			voiceState, ok := VoiceStates[i.GuildID]
			if !ok || voiceState == nil || voiceState.Streamer == nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "No audio is currently playing",
					},
				})
				return
			}

			if voiceState.Streamer.IsPaused() {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Audio is already paused",
					},
				})
				return
			}

			voiceState.Streamer.Pause()

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Paused audio playback",
				},
			})
		},
	})
}
