package bot

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session *discordgo.Session
	Token   string
	Prefix  string
}

func New(token string) (*Bot, error) {
	bot := &Bot{
		Token:  token,
		Prefix: "!",
	}

	return bot, nil
}

func (b *Bot) Start() error {
	session, err := discordgo.New("Bot " + b.Token)
	if err != nil {
		return fmt.Errorf("error creating Discord session: %w", err)
	}
	b.Session = session

	b.Session.AddHandler(b.messageHandler)

	err = b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	return nil
}

func (b *Bot) Stop() {
	if b.Session != nil {
		b.Session.Close()
	}
}

func (b *Bot) Run() error {
	if err := b.Start(); err != nil {
		return err
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

	// Clean shutdown
	b.Stop()
	return nil
}

func (b *Bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, b.Prefix) {
		command := strings.TrimPrefix(m.Content, b.Prefix)
		
		switch {
		case strings.HasPrefix(command, "ping"):
			s.ChannelMessageSend(m.ChannelID, "Pong!")
		case strings.HasPrefix(command, "hello"):
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Hello, %s!", m.Author.Username))
		}
	}
}
