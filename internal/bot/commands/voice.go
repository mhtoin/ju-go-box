package commands

import (
	audioplayer "github.com/mhtoin/ju-go-box/internal/audioplayer"
)

// VoiceState represents the state of voice playback for a guild
type VoiceState struct {
	StopChannel chan bool
	Streamer    *audioplayer.Streamer
}
