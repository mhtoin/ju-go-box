package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type VoiceState struct {
	VoiceConnection *discordgo.VoiceConnection
	IsPlaying       bool
	StopChannel     chan bool
}

func NewVoiceState() *VoiceState {
	return &VoiceState{
		IsPlaying:   false,
		StopChannel: make(chan bool),
	}
}

func (b *Bot) JoinVoiceChannel(guildId, channelId string) (*VoiceState, error) {
	voiceState := NewVoiceState()

	vc, err := b.Session.ChannelVoiceJoin(guildId, channelId, false, false)

	if err != nil {
		return nil, fmt.Errorf("error joining voice channel: %w", err)
	}

	voiceState.VoiceConnection = vc
	return voiceState, nil
}
