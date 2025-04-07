package commands

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
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
					Content: "Starting audio stream...",
				},
			})

			stop := make(chan bool)
			var wg sync.WaitGroup
			
			// Stream directly from yt-dlp to ffmpeg to get PCM data
			ytdlp := exec.Command("yt-dlp", 
				"-f", "bestaudio/best", // Get best audio format
				"--no-playlist",   // Don't process playlists
				"--extract-audio", // Extract only audio
				"--audio-format", "best", // Use best audio format
				"-o", "-",         // Output to stdout
				url)               // YouTube URL
			
			// Use ffmpeg with more controlled rate
			ffmpeg := exec.Command("ffmpeg",
				"-i", "pipe:0",      // Input from stdin
				"-f", "s16le",       // Format: signed 16-bit little-endian
				"-ar", "48000",      // Sample rate: 48kHz (Discord requirement)
				"-ac", "2",          // Channels: stereo
				"-b:a", "64k",       // Target bitrate: 64kbps
				"-af", "volume=0.5", // Lower volume slightly to avoid clipping
				"-bufsize", "2M",    // Increase buffer size
				"-maxrate", "64k",   // Max rate to avoid overflows
				"-application", "lowdelay", // Optimize for low delay
				"-threads", "2",     // Limit threads to prevent CPU overload
				"-loglevel", "warning", // Reduce ffmpeg log noise
				"pipe:1")            // Output to stdout
			
			ytdlpOut, err := ytdlp.StdoutPipe()
			if err != nil {
				log.Printf("Error creating ytdlp stdout pipe: %v", err)
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Error preparing audio stream: %v", err),
				})
				voiceConnection.Disconnect()
				return
			}
			
			// Also capture stderr for debugging
			ytdlpErr, err := ytdlp.StderrPipe()
			if err != nil {
				log.Printf("Error creating ytdlp stderr pipe: %v", err)
			} else {
				wg.Add(1)
				go func() {
					defer wg.Done()
					logOutput("yt-dlp", ytdlpErr)
				}()
			}
			
			ffmpeg.Stdin = ytdlpOut
			
			// Get ffmpeg output
			ffmpegOut, err := ffmpeg.StdoutPipe()
			if err != nil {
				log.Printf("Error creating ffmpeg stdout pipe: %v", err)
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Error preparing audio stream: %v", err),
				})
				voiceConnection.Disconnect()
				return
			}
			
			// Also capture stderr for debugging
			ffmpegErr, err := ffmpeg.StderrPipe()
			if err != nil {
				log.Printf("Error creating ffmpeg stderr pipe: %v", err)
			} else {
				wg.Add(1)
				go func() {
					defer wg.Done()
					logOutput("ffmpeg", ffmpegErr)
				}()
			}
			
			err = ytdlp.Start()
			if err != nil {
				log.Printf("Error starting ytdlp: %v", err)
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Error starting audio stream: %v", err),
				})
				voiceConnection.Disconnect()
				return
			}
			
			err = ffmpeg.Start()
			if err != nil {
				log.Printf("Error starting ffmpeg: %v", err)
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Error processing audio stream: %v", err),
				})
				ytdlp.Process.Kill()
				voiceConnection.Disconnect()
				return
			}
			
			// Notify user that streaming has started
			s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Now streaming audio from: %s", url),
			})
			
			// Initialize Opus encoder with settings optimized for music
			opusEncoder, err := gopus.NewEncoder(48000, 2, gopus.Audio)
			if err != nil {
				log.Printf("Error creating Opus encoder: %v", err)
				s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf("Error initializing audio encoder: %v", err),
				})
				ytdlp.Process.Kill()
				ffmpeg.Process.Kill()
				voiceConnection.Disconnect()
				return
			}
			
			// Set bitrate to Discord's preferred value (64kbps)
			opusEncoder.SetBitrate(64000)
			
			// Set speaking status to true
			err = voiceConnection.Speaking(true)
			if err != nil {
				log.Printf("Error setting speaking status: %v", err)
			}
			
			// Create a smaller buffer to avoid memory issues but large enough to handle jitter
			// 30 packets = ~600ms of audio
			audioBuffer := make(chan []byte, 30)
			
			// Signal for coordinating shutdown
			shutdownStarted := false
			var shutdownMutex sync.Mutex
			
			checkShutdown := func() bool {
				shutdownMutex.Lock()
				defer shutdownMutex.Unlock()
				return shutdownStarted
			}
			
			startShutdown := func() {
				shutdownMutex.Lock()
				defer shutdownMutex.Unlock()
				if !shutdownStarted {
					shutdownStarted = true
					close(stop)
				}
			}
			
			// Stream audio to Discord - use two goroutines
			// One to read and encode audio
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer startShutdown()
				defer func() {
					// Keep the buffer open until the player is done with it
					time.Sleep(200 * time.Millisecond)
					close(audioBuffer)
				}()
				
				// Buffer for PCM audio data - 960 frames * 2 channels * 2 bytes per sample
				// 960 frames is 20ms of audio at 48kHz
				pcmBuf := make([]int16, 960*2)
				
				errorCount := 0
				maxErrors := 5
				
				for {
					select {
					case <-stop:
						return
					default:
						if checkShutdown() {
							return
						}
						
						// Read raw PCM data from ffmpeg
						err := readPCMData(ffmpegOut, pcmBuf)
						if err != nil {
							errorCount++
							if err == io.EOF {
								log.Println("End of audio stream reached")
								return
							}
							
							log.Printf("Error reading from ffmpeg: %v", err)
							
							if errorCount >= maxErrors {
								log.Printf("Too many errors (%d), stopping playback", errorCount)
								return
							}
							
							// Small delay to avoid tight error loops
							time.Sleep(100 * time.Millisecond)
							continue
						}
						
						errorCount = 0
						
						// Encode PCM to Opus
						opusData, err := opusEncoder.Encode(pcmBuf, 960, 1000*2)
						if err != nil {
							log.Printf("Error encoding to Opus: %v", err)
							continue
						}
						
						// Only buffer if we're not shutting down
						if !checkShutdown() {
							select {
							case audioBuffer <- opusData:
								// Packet sent to buffer successfully
							case <-time.After(500 * time.Millisecond):
								// If buffer is full for too long, it means the consumer is stuck
								log.Println("Buffer send timeout, dropping packet")
							}
						}
					}
				}
			}()
			
			// Send audio to Discord in a controlled manner
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer startShutdown()
				defer func() {
					time.Sleep(500 * time.Millisecond)
					voiceConnection.Speaking(false)
					voiceConnection.Disconnect()
				}()
				
				// Discord expects packets every 20ms
				ticker := time.NewTicker(20 * time.Millisecond)
				defer ticker.Stop()
				
				bufferEmptyCount := 0
				skipLogThreshold := 20
				skipCount := 0
				
				for {
					if checkShutdown() {
						return
					}
					
					select {
					case <-stop:
						return
					case packet, ok := <-audioBuffer:
						if !ok {
							// Buffer closed, stream ended
							log.Println("Audio stream complete")
							return
						}
						
						// Wait for the next tick to maintain timing
						<-ticker.C
						
						// Send packet to Discord
						bufferEmptyCount = 0
						select {
						case voiceConnection.OpusSend <- packet:
							// Packet sent successfully
							skipCount = 0
						default:
							// Channel full, Discord can't keep up
							skipCount++
							if skipCount%skipLogThreshold == 0 {
								log.Printf("Discord voice channel buffer full, skipped %d packets", skipCount)
							}
						}
					case <-ticker.C:
						// No packet available, buffer is empty
						bufferEmptyCount++
						
						// After 250ms (12-13 ticks) of no data, check if we should stop
						if bufferEmptyCount > 12 {
							ytProc := ytdlp.Process
							ffProc := ffmpeg.Process
							
							ytRunning := ytProc != nil && isProcessRunning(ytProc.Pid)
							ffRunning := ffProc != nil && isProcessRunning(ffProc.Pid)
							
							if !ytRunning && !ffRunning {
								log.Println("Processes completed and buffer empty, ending playback")
								return
							}
							
							if bufferEmptyCount > 250 {
								log.Println("Buffer empty for too long, ending playback")
								return
							}
						}
					}
				}
			}()
			
			// Wait for processes to complete
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer startShutdown()
				
				ytdlpExitErr := ytdlp.Wait()
				if ytdlpExitErr != nil && !checkShutdown() {
					log.Printf("yt-dlp process exited with error: %v", ytdlpExitErr)
				}
				
				ffmpegExitErr := ffmpeg.Wait()
				if ffmpegExitErr != nil && !checkShutdown() {
					log.Printf("ffmpeg process exited with error: %v", ffmpegExitErr)
				}
				
				log.Println("Both processes completed")
			}()
			
			wg.Wait()
			log.Println("Playback completed")
		},
	})
}

// readPCMData reads PCM data from the given reader into the buffer
func readPCMData(reader io.Reader, buf []int16) error {
	byteBuffer := make([]byte, len(buf)*2)
	
	// Read raw PCM bytes
	readBytes, err := io.ReadFull(reader, byteBuffer)
	if err != nil {
		return err
	}
	
	if readBytes != len(byteBuffer) {
		return fmt.Errorf("incomplete read: got %d bytes, expected %d", readBytes, len(byteBuffer))
	}
	
	// Convert bytes to int16 samples
	for i := 0; i < len(buf); i++ {
		buf[i] = int16(byteBuffer[i*2]) | int16(byteBuffer[i*2+1])<<8
	}
	
	return nil
}

// logOutput reads from a reader and logs output
func logOutput(prefix string, reader io.Reader) {
	scanner := io.LimitReader(reader, 1024*1024) // Limit to 1MB of logs
	buf := make([]byte, 1024)
	
	for {
		n, err := scanner.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("%s logger error: %v", prefix, err)
			}
			return
		}
		
		if n > 0 {
			log.Printf("[%s] %s", prefix, string(buf[:n]))
		}
	}
}

// isProcessRunning checks if a process with the given PID is still running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// On Unix systems, FindProcess always succeeds, so we need to send signal 0
	// to check if the process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}