package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"telegramBot/internal/bot"
	"telegramBot/internal/config"
)

func main() {
	// Configure logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Telegram Torrent Bot...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and initialize the bot
	torrentBot, err := bot.NewBot(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// Setup graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Start the bot in a goroutine
	go func() {
		if err := torrentBot.Start(); err != nil {
			log.Fatalf("Bot error: %v", err)
		}
	}()

	log.Println("Bot is now running. Press CTRL-C to exit.")

	// Wait for termination signal
	<-signals
	log.Println("Shutting down gracefully...")
}
