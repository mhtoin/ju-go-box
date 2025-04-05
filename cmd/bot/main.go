package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mhtoin/ju-go-box/internal/bot"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

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

