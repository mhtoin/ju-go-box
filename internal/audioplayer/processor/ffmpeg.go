package processor

import (
	"io"
	"os/exec"
)

type FfmpegProcessor struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func NewFfmpegProcessor() *FfmpegProcessor {
	return &FfmpegProcessor{}
}

func (p *FfmpegProcessor) Process(r io.Reader, w io.Writer) error {
	p.cmd = exec.Command("ffmpeg",
		"-i", "pipe:0", // Input from stdin
		"-f", "s16le", // Format: signed 16-bit little-endian
		"-ar", "48000", // Sample rate: 48kHz (Discord requirement)
		"-ac", "2", // Channels: stereo
		"-b:a", "64k", // Target bitrate: 64kbps
		"-af", "volume=0.5", // Lower volume slightly to avoid clipping
		"-bufsize", "2M", // Increase buffer size
		"-maxrate", "64k", // Max rate to avoid overflows
		"-application", "lowdelay", // Optimize for low delay
		"-threads", "2", // Limit threads to prevent CPU overload
		"-loglevel", "warning", // Reduce ffmpeg log noise
		"pipe:1") // Output to stdout

	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		return err
	}

	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := p.cmd.Start(); err != nil {
		return err
	}

	go io.Copy(p.stdin, r)
	go io.Copy(w, p.stdout)

	return nil
}

func (p *FfmpegProcessor) Stop() error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}
