package main

import (
	"log"
	"os"

	"github.com/mhtoin/ju-go-box/internal/bot"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}

	discordBot, err := bot.New(token)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	if err := discordBot.Run(); err != nil {
		log.Fatalf("Error running bot: %v", err)
	}
}

