package source

import (
	"io"
	"os/exec"
)

type YoutubeSource struct {
	url    string
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

func NewYoutubeSource(url string) *YoutubeSource {
	return &YoutubeSource{
		url: url,
	}
}

func (y *YoutubeSource) Stream(w io.Writer) error {
	y.cmd = exec.Command("yt-dlp",
		"-f", "bestaudio/best",
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "best",
		"-o", "-",
		y.url)

	stdout, err := y.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := y.cmd.Start(); err != nil {
		return err
	}

	go io.Copy(w, stdout)

	return nil
}

func (y *YoutubeSource) Stop() error {
	if y.cmd != nil && y.cmd.Process != nil {
		return y.cmd.Process.Kill()
	}
	return nil
}
