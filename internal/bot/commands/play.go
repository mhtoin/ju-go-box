package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
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
			
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Downloading and preparing audio...",
				},
			})

			
			outputFile := fmt.Sprintf("%s/audio.opus", "./data/")

			defer os.Remove(outputFile)

			/*
			* Proof of concept: download the audio file and play it
			*/

			cmd := exec.Command("yt-dlp", 
				"--extract-audio",           // Extract audio only
				"--audio-format", "opus",    // Convert to opus format
				"--audio-quality", "0",      // Best quality
				"-o", outputFile,            // Output file
				url)                         // YouTube URL

			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("Error running yt-dlp: %v, output: %s", err, string(output))
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Error downloading audio: %v", err),
				})
				return
			}

			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Audio downloaded successfully from: %s", url),
			})
			fmt.Println("Audio downloaded successfully", outputFile)

			dgvoice.PlayAudioFile(voiceConnection, outputFile, make(chan bool))

			voiceConnection.Speaking(false)
			voiceConnection.Disconnect()
		},
	})
}