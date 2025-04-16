# Telegram Torrent Bot

Telegram Torrent Bot is a Go-based application that allows users to manage torrents via a Telegram bot. It integrates with qBittorrent and supports multiple trackers for downloading torrents.

## Features

- Manage torrents via Telegram commands.
- Integration with qBittorrent for torrent management.
- Support for multiple trackers (e.g., Rutracker, Kinozal).
- Categorize torrents into predefined categories (e.g., Movies, TV Shows, Games).
- Restrict access to specific Telegram users.

## Project Structure

```bash
telegramDocker/
├── Dockerfile.multistage   # Dockerfile for building and deploying the application
├── env.fish               # Environment variables for Fish shell
├── env.list               # Environment variables for Docker
├── go.mod                 # Go module dependencies
├── go.sum                 # Go module checksums
├── README.md              # Project documentation
├── words.txt              # Additional configuration or data file
├── cmd/                   # Main application entry point
│   └── bot/               # Telegram bot implementation
│       └── main.go        # Main file for the bot
├── internal/              # Internal application logic
│   ├── bot/               # Bot-related logic (commands, keyboard, etc.)
│   ├── client/            # Client integrations (e.g., qBittorrent, trackers)
│   ├── config/            # Configuration loading and management
│   ├── models/            # Data models
│   └── utils/             # Utility functions
```

## Prerequisites

- Go 1.19 or later
- Docker (for containerized deployment)
- qBittorrent with WebUI enabled

## Setup

1. Clone the repository:

   ```bash
   git clone <repository-url>
   cd telegramDocker
   ```

2. Configure environment variables:
   - Update `env.list` with your configuration.
   - Example:

     ```bash
     TELEGRAMBOTAPI=<your_telegram_bot_api>
     QBITTORRENT_URL=http://localhost:8080
     TORRENTUSER=admin
     TORRENTPASSWORD=adminpassword
     RUTRACKERUSER=<your_rutracker_user>
     RUTRACKERPASSWORD=<your_rutracker_password>
     ALLOWED_USERS=123456789|987654321
     ```

3. Build and run the application using Docker:

   ```bash
   docker build -t telegramdocker:latest -f Dockerfile.multistage .
   docker run --env-file env.list telegramdocker:latest
   ```

## Usage

- Start the Telegram bot and send commands to manage torrents.
- Supported commands include adding torrents, listing torrents, and more.

## Development

1. Install dependencies:

   ```bash
   go mod download
   ```

2. Run the application locally:

   ```bash
   go run cmd/bot/main.go
   ```

3. Run tests:

   ```bash
   go test ./...
   ```
