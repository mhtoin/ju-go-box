package commands

// VoiceState represents the state of voice playback for a guild
type VoiceState struct {
	StopChannel chan bool
} 