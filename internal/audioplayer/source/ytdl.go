package source

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
)

type YoutubeSource struct {
	url    string
	cmd    *exec.Cmd
	stdout io.ReadCloser
	title  string
}

func NewYoutubeSource(url string) *YoutubeSource {
	return &YoutubeSource{
		url:   url,
		title: "",
	}
}

func (y *YoutubeSource) GetTitle() (string, error) {
	cmd := exec.Command("yt-dlp", "--get-title", y.url)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (y *YoutubeSource) Stream(w io.Writer) error {
	y.cmd = exec.Command("yt-dlp",
		"-f", "bestaudio/best",
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "best",
		"-o", "-",
		y.url)

	var err error
	y.stdout, err = y.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := y.cmd.Start(); err != nil {
		return err
	}

	go io.Copy(w, y.stdout)
	y.title, err = y.GetTitle()
	if err != nil {
		return err
	}

	return nil
}

func (y *YoutubeSource) Stop() error {
	if y.cmd != nil && y.cmd.Process != nil {
		return y.cmd.Process.Kill()
	}
	return nil
}
