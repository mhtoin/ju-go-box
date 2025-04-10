package bot

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/mhtoin/ju-go-box/internal/bot/commands"
)

type Bot struct {
	Session *discordgo.Session
	Token   string
}

func New(token string) (*Bot, error) {
	bot := &Bot{
		Token: token,
	}

	return bot, nil
}

func (b *Bot) Start() error {
	session, err := discordgo.New("Bot " + b.Token)
	if err != nil {
		return fmt.Errorf("error creating Discord session: %w", err)
	}
	b.Session = session

	b.Session.AddHandler(b.interactionHandler)

	err = b.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	appCommands := commands.GetApplicationCommands()
	for _, cmd := range appCommands {
		_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("error creating command %s: %w", cmd.Name, err)
		}
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

	b.Stop()
	return nil
}

func (b *Bot) interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if cmd, ok := commands.Commands[i.ApplicationCommandData().Name]; ok {
		cmd.Handler(s, i)
	}
}

func (b *Bot) UpdateStatus(status string, activityType discordgo.ActivityType, activityName string) error {
	activity := discordgo.Activity{
		Name: activityName,
		Type: activityType,
	}

	updateData := discordgo.UpdateStatusData{
		Activities: []*discordgo.Activity{&activity},
		Status:     status,
		AFK:        false,
	}

	return b.Session.UpdateStatusComplex(updateData)
}
