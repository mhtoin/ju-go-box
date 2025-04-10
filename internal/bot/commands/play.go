package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	audioplayer "github.com/mhtoin/ju-go-box/internal/audioplayer"
)

func init() {
	RegisterCommand(Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "play",
			Description: "Play audio from a YouTube link",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "url",
					Description: "The YouTube URL to play",
					Required:    true,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			url := i.ApplicationCommandData().Options[0].StringValue()

			voiceState, err := s.State.VoiceState(i.GuildID, i.Member.User.ID)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You need to be in a voice channel to use this command!",
					},
				})
				return
			}

			voiceConnection, err := s.ChannelVoiceJoin(i.GuildID, voiceState.ChannelID, false, false)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Error joining voice channel: %v", err),
					},
				})
				return
			}

			// Create a new voice state for global control
			vs := &VoiceState{
				StopChannel: make(chan bool),
			}
			VoiceStates[i.GuildID] = vs

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Starting audio stream...",
				},
			})

			streamer := audioplayer.NewStreamer(voiceConnection)
			vs.Streamer = streamer
			err = streamer.Stream(url)

			if err != nil {
				log.Printf("Error starting stream: %v", err)
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Error starting audio stream: %v", err),
				})
				voiceConnection.Disconnect()
				return
			}

			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Now streaming audio from: %s", url),
			})

			go func() {
				<-vs.StopChannel
				streamer.Stop()
				voiceConnection.Disconnect()
			}()

		},
	})
}
