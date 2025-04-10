package audio

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mhtoin/ju-go-box/internal/audioplayer/processor"
	"github.com/mhtoin/ju-go-box/internal/audioplayer/source"
	"layeh.com/gopus"
)

type Streamer struct {
	source     source.Source
	processor  processor.Processor
	connection *discordgo.VoiceConnection
	stopChan   chan bool
	buffer     chan []byte
	wg         sync.WaitGroup
	pauseChan  chan bool
	isPaused   bool
	pauseMutex sync.Mutex
}

func NewStreamer(vc *discordgo.VoiceConnection) *Streamer {
	return &Streamer{
		connection: vc,
		stopChan:   make(chan bool),
		buffer:     make(chan []byte, 30),
		pauseChan:  make(chan bool),
		isPaused:   false,
	}
}

func (s *Streamer) Stream(url string) error {
	s.source = source.NewYoutubeSource(url)
	s.processor = processor.NewFfmpegProcessor()

	sourceToProcessorReader, sourceToProcessorWriter := io.Pipe()

	if err := s.source.Stream(sourceToProcessorWriter); err != nil {
		return err
	}

	processorToEncoderReader, processorToEncoderWriter := io.Pipe()
	if err := s.processor.Process(sourceToProcessorReader, processorToEncoderWriter); err != nil {
		s.source.Stop()
		return err
	}

	opusEncoder, err := gopus.NewEncoder(48000, 2, gopus.Audio)
	if err != nil {
		s.source.Stop()
		s.processor.Stop()
		return err
	}

	if err := s.connection.Speaking(true); err != nil {
		log.Printf("Error setting speaking status: %v", err)
	}

	s.wg.Add(1)
	go s.encodeAndBuffer(processorToEncoderReader, opusEncoder)

	s.wg.Add(1)
	go s.streamToDiscord()

	return nil
}

func (s *Streamer) encodeAndBuffer(r io.Reader, e *gopus.Encoder) {
	defer s.wg.Done()
	defer close(s.buffer)

	pcmBuffer := make([]int16, 960*2)
	byteBuffer := make([]byte, len(pcmBuffer)*2)

	for {
		select {
		case <-s.stopChan:
			return
		default:
			readBytes, err := io.ReadFull(r, byteBuffer)
			if err != nil {
				if err == io.EOF {
					log.Println("End of audio stream reached")
					return
				}
				log.Printf("Error reading PCM data: %v", err)
				continue
			}

			if readBytes != len(byteBuffer) {
				log.Printf("Incomplete read: got %d bytes, expected %d", readBytes, len(byteBuffer))
				continue
			}

			for i := 0; i < len(pcmBuffer); i++ {
				pcmBuffer[i] = int16(byteBuffer[i*2]) | int16(byteBuffer[i*2+1])<<8
			}

			opusData, err := e.Encode(pcmBuffer, 960, 1000*2)
			if err != nil {
				log.Printf("Error encoding to Opus: %v", err)
				continue
			}

			select {
			case s.buffer <- opusData:
			case <-s.stopChan:
				return
			case <-time.After(500 * time.Millisecond):
				s.pauseMutex.Lock()
				isPaused := s.isPaused
				s.pauseMutex.Unlock()
				if !isPaused {
					log.Println("Buffer send timeout, dropping packet")
				}
			}
		}
	}
}

func (s *Streamer) streamToDiscord() {
	defer s.wg.Done()
	defer s.connection.Speaking(false)

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	bufferEmptyCount := 0

	for {
		s.pauseMutex.Lock()
		paused := s.isPaused
		s.pauseMutex.Unlock()

		if paused {
			s.connection.Speaking(false)

			select {
			case <-s.pauseChan:
				s.connection.Speaking(true)
				continue
			case <-s.stopChan:
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}

		select {
		case <-s.stopChan:
			return
		case <-s.pauseChan:
			s.pauseMutex.Lock()
			s.isPaused = true
			s.pauseMutex.Unlock()
			continue
		case packet, ok := <-s.buffer:
			if !ok {
				log.Println("Audio stream complete")
				return
			}

			<-ticker.C
			bufferEmptyCount = 0
			select {
			case s.connection.OpusSend <- packet:
			case <-s.stopChan:
				return
			case <-s.pauseChan:
				s.pauseMutex.Lock()
				s.isPaused = true
				s.pauseMutex.Unlock()
				continue
			default:
				log.Println("Discord voice channel buffer full, skipping packet")
			}

		case <-ticker.C:
			bufferEmptyCount++
			if bufferEmptyCount > 250 {
				log.Println("No audio data received for 200ms, pausing to prevent buffer overflow")
				return
			}
		}
	}
}

func (s *Streamer) Pause() {
	s.pauseMutex.Lock()
	defer s.pauseMutex.Unlock()

	if !s.isPaused {
		s.isPaused = true
		select {
		case s.pauseChan <- true:
			log.Println("Stream paused")
		default:
			log.Println("Failed to send pause signal")
		}
	}
}

func (s *Streamer) Resume() {
	s.pauseMutex.Lock()
	defer s.pauseMutex.Unlock()

	if s.isPaused {
		s.isPaused = false
		select {
		case s.pauseChan <- true:
			log.Println("Stream resumed")
		default:
			log.Println("Failed to send resume signal")
		}
	}
}

func (s *Streamer) IsPaused() bool {
	s.pauseMutex.Lock()
	defer s.pauseMutex.Unlock()
	return s.isPaused
}

func (s *Streamer) GetTitle() string {
	if ys, ok := s.source.(*source.YoutubeSource); ok {
		title, _ := ys.GetTitle()
		return title
	}
	return "Unknown"
}

func (s *Streamer) Stop() {
	close(s.stopChan)

	if s.source != nil {
		s.source.Stop()
	}

	if s.processor != nil {
		s.processor.Stop()
	}

	s.wg.Wait()
}
