package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"telegramBot/internal/models"
)

// Config holds all application configuration
type Config struct {
	TelegramBotToken   string
	QBittorrent        models.QBittorrentCredentials
	TrackerCredentials map[string]models.TrackerCredentials
	TorrentCategories  map[string]models.TorrentCategory
	AllowedUsers       []int64
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	botToken := os.Getenv("TELEGRAMBOTAPI")
	if botToken == "" {
		return nil, errors.New("TELEGRAMBOTAPI environment variable not set")
	}

	qbtURL := os.Getenv("QBITTORRENT_URL")
	if qbtURL == "" {
		qbtURL = "http://localhost:8080" // Default qBittorrent WebUI URL
	}
	var allowedUsersList []int64

	allowUsers := os.Getenv("ALLOWED_USERS")
	if allowUsers != "" {
		users := strings.SplitSeq(allowUsers, "|")
		for user := range users {
			userID, err := strconv.ParseInt(user, 10, 64)
			if err != nil {
				return nil, errors.New("invalid user ID in ALLOWED_USERS")
			}
			allowedUsersList = append(allowedUsersList, userID)
		}
	}

	config := &Config{
		TelegramBotToken: botToken,
		QBittorrent: models.QBittorrentCredentials{
			URL:      qbtURL,
			Username: os.Getenv("TORRENTUSER"),
			Password: os.Getenv("TORRENTPASSWORD"),
		},
		TrackerCredentials: map[string]models.TrackerCredentials{
			"rutracker": {
				LoginURL: "https://rutracker.org/forum/login.php",
				Username: os.Getenv("RUTRACKERUSER"),
				Password: os.Getenv("RUTRACKERPASSWORD"),
				LoginKey: os.Getenv("RUTRACKERLOGIN"),
				FormData: map[string]string{
					"login_username": os.Getenv("RUTRACKERUSER"),
					"login_password": os.Getenv("RUTRACKERPASSWORD"),
					"login":          os.Getenv("RUTRACKERLOGIN"),
				},
			},
			"kinozal": {
				LoginURL: "https://kinozal.tv/takelogin.php",
				Username: os.Getenv("KINOZALUSER"),
				Password: os.Getenv("KINOZALPASSWORD"),
				FormData: map[string]string{
					"username": os.Getenv("KINOZALUSER"),
					"password": os.Getenv("KINOZALPASSWORD"),
				},
			},
		},
		TorrentCategories: map[string]models.TorrentCategory{
			"Movies.": {
				Name:     "Movies.",
				SavePath: os.Getenv("MOVIES_PATH"),
				Callback: "Movies.",
			},
			"TV Shows.": {
				Name:     "TV Shows.",
				SavePath: os.Getenv("TV_SHOWS_PATH"),
				Callback: "TV Shows.",
			},
			"Games.": {
				Name:     "Games.",
				SavePath: os.Getenv("GAMES_PATH"),
				Callback: "Games.",
			},
			"MultiParts.": {
				Name:     "MultiParts.",
				SavePath: os.Getenv("MULTIPARTS_PATH"),
				Callback: "MultiParts.",
			},
			"AudioBooks.": {
				Name:     "AudioBooks.",
				SavePath: os.Getenv("AUDIOBOOKS_PATH"),
				Callback: "AudioBooks.",
			},
			"MANGA.": {
				Name:     "MANGA.",
				SavePath: os.Getenv("MANGA_PATH"),
				Callback: "MANGA.",
			},
			"COMICS.": {
				Name:     "COMICS.",
				SavePath: os.Getenv("COMICS_PATH"),
				Callback: "COMICS.",
			},
		},
		AllowedUsers: allowedUsersList,
	}

	// Set defaults for save paths if not provided in environment variables
	if category, ok := config.TorrentCategories["Movies"]; ok && category.SavePath == "" {
		category.SavePath = "Z:\\"
		config.TorrentCategories["Movies"] = category
	}
	if category, ok := config.TorrentCategories["TV Shows"]; ok && category.SavePath == "" {
		category.SavePath = "Y:\\"
		config.TorrentCategories["TV Shows"] = category
	}
	if category, ok := config.TorrentCategories["Games"]; ok && category.SavePath == "" {
		category.SavePath = "D:\\gamesTorrent"
		config.TorrentCategories["Games"] = category
	}
	if category, ok := config.TorrentCategories["MultiParts"]; ok && category.SavePath == "" {
		category.SavePath = "W:\\"
		config.TorrentCategories["MultiParts"] = category
	}
	if category, ok := config.TorrentCategories["AudioBooks"]; ok && category.SavePath == "" {
		category.SavePath = "T:\\"
		config.TorrentCategories["AudioBooks"] = category
	}
	if category, ok := config.TorrentCategories["MANGA"]; ok && category.SavePath == "" {
		category.SavePath = "R:\\"
		config.TorrentCategories["MANGA"] = category
	}
	if category, ok := config.TorrentCategories["COMICS"]; ok && category.SavePath == "" {
		category.SavePath = "S:\\"
		config.TorrentCategories["COMICS"] = category
	}

	return config, nil
}
