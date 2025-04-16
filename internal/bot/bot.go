package bot

import (
	"fmt"
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"telegramBot/internal/client"
	"telegramBot/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot represents the Telegram bot
type Bot struct {
	api              *tgbotapi.BotAPI
	config           *config.Config
	qbtClient        *client.QBittorrentClient
	trackerClient    *client.TorrentTrackerClient
	torrentLinkRegex *regexp.Regexp
	trackerRegex     *regexp.Regexp
	pendingLinks     map[int64]string
}

// NewBot creates a new instance of the Telegram bot
func NewBot(config *config.Config) (*Bot, error) {
	// Initialize Telegram bot
	bot, err := tgbotapi.NewBotAPI(config.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	// Initialize qBittorrent client
	qbtClient, err := client.NewQBittorrentClient(config.QBittorrent)
	if err != nil {
		return nil, fmt.Errorf("failed to create qBittorrent client: %w", err)
	}

	// Initialize torrent tracker client
	trackerClient, err := client.NewTorrentTrackerClient(config.TrackerCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracker client: %w", err)
	}

	// Compile regex patterns
	torrentLinkRegex := regexp.MustCompile(`(http|https)://(kinozal|rutracker)\.[a-z]{2,4}\b([-a-zA-Z0-9@:%_+.~#?&/=]*)`)
	trackerRegex := regexp.MustCompile("kinozal|rutracker")

	return &Bot{
		api:              bot,
		config:           config,
		qbtClient:        qbtClient,
		trackerClient:    trackerClient,
		torrentLinkRegex: torrentLinkRegex,
		trackerRegex:     trackerRegex,
		pendingLinks:     make(map[int64]string),
	}, nil
}

// Start starts the bot and listens for updates
func (b *Bot) Start() error {
	// Set update config
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// Get updates channel
	updates := b.api.GetUpdatesChan(updateConfig)

	// Log bot info
	log.Printf("Authorized on account %s", b.api.Self.UserName)

	// Process updates
	for update := range updates {
		go b.handleUpdate(update)
	}

	return nil
}

// handleUpdate processes a single update from Telegram
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	// Handle callback queries (button presses)
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
		return
	}

	// Ignore updates without messages
	if update.Message == nil {
		return
	}

	// Try to match torrent links in messages
	if b.torrentLinkRegex.MatchString(update.Message.Text) {
		b.handleTorrentLink(update.Message)
		return
	}

	// Handle commands
	if update.Message.IsCommand() {
		b.handleCommand(update.Message)
		return
	}
}

// handleCallbackQuery processes callbacks from inline keyboards
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	// Extract callback data
	data := query.Data
	chatID := query.Message.Chat.ID
	messageID := query.Message.MessageID

	// Acknowledge the callback query
	callback := tgbotapi.NewCallback(query.ID, "")
	b.api.Request(callback)

	// Handle torrent category selection (for downloads)
	if strings.HasSuffix(data, ".") && b.pendingLinks[chatID] != "" {
		torrentLink := b.pendingLinks[chatID]
		b.handleTorrentDownload(chatID, messageID, torrentLink, data)
		return
	}

	// Handle torrent management actions
	if strings.Contains(data, ":") {
		parts := strings.Split(data, ":")
		if len(parts) < 2 {
			b.sendErrorMessage(chatID, "Invalid callback data")
			return
		}

		action := parts[0]

		switch action {
		case "manage":
			// Show detailed info for a specific torrent
			// Check if page is specified
			if len(parts) > 2 && parts[2] == "page" && len(parts) > 3 {
				page, _ := strconv.Atoi(parts[3])
				b.handleTorrentDetails(chatID, messageID, parts[1], page)
			} else {
				b.handleTorrentDetails(chatID, messageID, parts[1], 0)
			}
		case "pause", "resume", "delete", "deletewithdata", "info":
			// Perform actions on a specific torrent
			b.handleTorrentAction(chatID, messageID, action, parts[1])
		case "list":
			// Handle list pagination
			if len(parts) > 2 && parts[1] == "page" {
				page, _ := strconv.Atoi(parts[2])
				b.handleListPagination(chatID, messageID, page)
			}
		default:
			b.sendErrorMessage(chatID, "Unknown action")
		}
		return
	}

	// Unknown callback data
	b.sendErrorMessage(chatID, "Unknown callback data")
}

// handleTorrentLink processes a message containing a torrent link
func (b *Bot) handleTorrentLink(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Store the link for later processing
	b.pendingLinks[chatID] = message.Text

	// Send category selection keyboard
	msg := tgbotapi.NewMessage(chatID, "What category should this download be saved as?")
	msg.ReplyMarkup = CreateCategoryKeyboard(b.config.TorrentCategories)

	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Error sending category keyboard: %v", err)
		b.sendErrorMessage(chatID, "Failed to send keyboard")
	}
}

// handleTorrentDownload processes a torrent download request after category selection
func (b *Bot) handleTorrentDownload(chatID int64, messageID int, torrentLink, categoryKey string) {
	// Edit the message to show processing
	edit := tgbotapi.NewEditMessageText(chatID, messageID, "Processing download request...")
	edit.ReplyMarkup = nil
	b.api.Send(edit)

	// Get category save path
	category, exists := b.config.TorrentCategories[categoryKey]
	if !exists {
		b.sendErrorMessage(chatID, "Invalid category selected")
		return
	}

	// Extract tracker and ID from link
	trackerName, id, err := ProcessTorrentLink(torrentLink)
	if err != nil {
		b.sendErrorMessage(chatID, fmt.Sprintf("Error processing link: %v", err))
		return
	}

	// Download and add torrent
	result, err := DownloadAndAddTorrent(b.trackerClient, b.qbtClient, trackerName, id, category.SavePath)
	if err != nil {
		b.sendErrorMessage(chatID, fmt.Sprintf("Download failed: %v", err))
		return
	}

	// Update message with success
	edit = tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf("✅ %s\n\nSave path: %s", result, category.SavePath))
	b.api.Send(edit)

	// Clear the pending link
	delete(b.pendingLinks, chatID)
}

// handleCommand processes bot commands
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	command := message.Command()
	command = strings.ToLower(command)
	args := message.CommandArguments()

	if !slices.Contains(b.config.AllowedUsers, chatID) {
		msg := tgbotapi.NewMessage(chatID, "You are not authorized to use this bot.")
		b.api.Send(msg)
		return
	}

	switch command {
	case "start", "help":
		b.handleHelpCommand(chatID)
	case "status":
		b.handleStatusCommand(chatID)
	case "torrent":
		b.handleTorrentCommand(chatID, args)
	case "list":
		b.handleListCommand(chatID)
	case "reconnect":
		b.handleReconnectCommand(chatID)
	case "password":
		b.handlePasswordCommand(chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, "Unknown command. Type /help for available commands.")
		b.api.Send(msg)
	}
}

// handleHelpCommand sends help information
func (b *Bot) handleHelpCommand(chatID int64) {
	helpText := `*Torrent Bot Commands:*

/status - Show status of all torrents
/torrent [name] - Search for torrents by name
/list - Show a list of active torrents
/password - Generate a random password

*Other Features:*
- Send a link from a supported tracker to download it
- Use buttons to manage your torrents

*Supported Trackers:*
- RuTracker
- Kinozal`

	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// handleStatusCommand shows the status of all torrents
func (b *Bot) handleStatusCommand(chatID int64) {
	status, keyboard, err := HandleTorrentStatus(b.qbtClient, 0)
	if err != nil {
		// Try to reconnect and retry
		if b.tryReconnect(chatID, "getting torrent status") {
			// Retry after successful reconnection
			status, keyboard, err = HandleTorrentStatus(b.qbtClient, 0)
			if err != nil {
				b.sendErrorMessage(chatID, fmt.Sprintf("Error getting status even after reconnection: %v", err))
				return
			}
		} else {
			// Reconnection failed
			return
		}
	}

	msg := tgbotapi.NewMessage(chatID, status)
	msg.ParseMode = "Markdown"
	if len(keyboard.InlineKeyboard) > 0 {
		msg.ReplyMarkup = keyboard
	}
	b.api.Send(msg)
}

// tryReconnect attempts to reconnect to qBittorrent and returns whether it was successful
func (b *Bot) tryReconnect(chatID int64, operation string) bool {
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ Lost connection to qBittorrent while %s. Attempting to reconnect...", operation))
	sentMsg, _ := b.api.Send(msg)

	// Attempt to reconnect
	err := b.qbtClient.Reconnect()
	if err != nil {
		edit := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID,
			fmt.Sprintf("❌ Failed to reconnect: %v", err))
		b.api.Send(edit)
		return false
	}

	// Test the connection
	_, err = b.qbtClient.GetTorrents("")
	if err != nil {
		edit := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID,
			fmt.Sprintf("❌ Reconnection failed during testing: %v", err))
		b.api.Send(edit)
		return false
	}

	// Update message with success
	edit := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID,
		"✅ Successfully reconnected to qBittorrent. Retrying operation...")
	b.api.Send(edit)
	return true
}

// handleTorrentCommand shows details for specific torrents
func (b *Bot) handleTorrentCommand(chatID int64, args string) {
	if args == "" {
		msg := tgbotapi.NewMessage(chatID, "Please provide a torrent name to search for. Example: /torrent ubuntu")
		b.api.Send(msg)
		return
	}

	text, keyboard, err := HandleSpecificTorrentStatus(b.qbtClient, args)
	if err != nil {
		// Try to reconnect and retry
		if b.tryReconnect(chatID, "searching for torrents") {
			// Retry after successful reconnection
			text, keyboard, err = HandleSpecificTorrentStatus(b.qbtClient, args)
			if err != nil {
				b.sendErrorMessage(chatID, fmt.Sprintf("Error even after reconnection: %v", err))
				return
			}
		} else {
			// Reconnection failed
			return
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if len(keyboard.InlineKeyboard) > 0 {
		msg.ReplyMarkup = keyboard
	}
	b.api.Send(msg)
}

// handleListCommand shows a list of torrents with management options
func (b *Bot) handleListCommand(chatID int64) {
	torrents, err := b.qbtClient.GetTorrents("")
	if err != nil {
		// Try to reconnect and retry
		if b.tryReconnect(chatID, "listing torrents") {
			// Retry after successful reconnection
			torrents, err = b.qbtClient.GetTorrents("")
			if err != nil {
				b.sendErrorMessage(chatID, fmt.Sprintf("Error getting torrent list even after reconnection: %v", err))
				return
			}
		} else {
			// Reconnection failed
			return
		}
	}

	if len(torrents) == 0 {
		msg := tgbotapi.NewMessage(chatID, "No torrents found")
		b.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Select a torrent to manage:")
	msg.ReplyMarkup = CreateTorrentListKeyboard(torrents, 20, 0)
	b.api.Send(msg)
}

// handlePasswordCommand generates a random password
func (b *Bot) handlePasswordCommand(chatID int64) {
	password, err := HandlePasswordCommand("words.txt")
	if err != nil {
		b.sendErrorMessage(chatID, fmt.Sprintf("Error generating password: %v", err))
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Generated password: `%s`", password))
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// handleTorrentDetails shows detailed information for a specific torrent
func (b *Bot) handleTorrentDetails(chatID int64, messageID int, hash string, page int) {
	text, keyboard, err := HandleSpecificTorrentStatus(b.qbtClient, "manage:"+hash)
	if err != nil {
		// Send a temporary message about the error
		tempMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ Error accessing torrent details: %v", err))
		sentMsg, _ := b.api.Send(tempMsg)

		// Try to reconnect
		if b.tryReconnect(chatID, "getting torrent details") {
			// Delete the temporary message
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
			b.api.Request(deleteMsg)

			// Retry after successful reconnection
			text, keyboard, err = HandleSpecificTorrentStatus(b.qbtClient, "manage:"+hash)
			if err != nil {
				b.sendErrorMessage(chatID, fmt.Sprintf("Error even after reconnection: %v", err))
				return
			}
		} else {
			// Reconnection failed
			return
		}
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	if len(keyboard.InlineKeyboard) > 0 {
		edit.ReplyMarkup = &keyboard
	}
	b.api.Send(edit)
}

// handleTorrentAction performs actions on a specific torrent
func (b *Bot) handleTorrentAction(chatID int64, messageID int, action, hash string) {
	text, keyboard, err := HandleTorrentAction(b.qbtClient, action, hash)
	if err != nil {
		// Send a temporary message about the error
		tempMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⚠️ Error performing action %s: %v", action, err))
		sentMsg, _ := b.api.Send(tempMsg)

		// Try to reconnect
		if b.tryReconnect(chatID, "performing torrent action") {
			// Delete the temporary message
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
			b.api.Request(deleteMsg)

			// Retry after successful reconnection
			text, keyboard, err = HandleTorrentAction(b.qbtClient, action, hash)
			if err != nil {
				b.sendErrorMessage(chatID, fmt.Sprintf("Error even after reconnection: %v", err))
				return
			}
		} else {
			// Reconnection failed
			return
		}
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	if len(keyboard.InlineKeyboard) > 0 {
		edit.ReplyMarkup = &keyboard
	}
	b.api.Send(edit)
}

// handleReconnectCommand forces a reconnection to qBittorrent
func (b *Bot) handleReconnectCommand(chatID int64) {
	// Send a message indicating we're attempting to reconnect
	msg := tgbotapi.NewMessage(chatID, "🔄 Attempting to reconnect to qBittorrent...")
	sentMsg, _ := b.api.Send(msg)

	// Attempt to reconnect
	err := b.qbtClient.Reconnect()
	if err != nil {
		// Update message with error
		edit := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID,
			fmt.Sprintf("❌ Failed to reconnect: %v", err))
		b.api.Send(edit)
		return
	}

	// Test the connection
	_, err = b.qbtClient.GetTorrents("")
	if err != nil {
		// Update message with error
		edit := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID,
			fmt.Sprintf("❌ Reconnection failed during testing: %v", err))
		b.api.Send(edit)
		return
	}

	// Update message with success
	edit := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID,
		"✅ Successfully reconnected to qBittorrent")
	b.api.Send(edit)
}

// sendErrorMessage sends an error message to the user
func (b *Bot) sendErrorMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, "❌ "+text)
	b.api.Send(msg)
}

// handleListPagination handles pagination for the torrent list
func (b *Bot) handleListPagination(chatID int64, messageID int, page int) {
	torrents, err := b.qbtClient.GetTorrents("")
	if err != nil {
		b.sendErrorMessage(chatID, fmt.Sprintf("Error getting torrent list: %v", err))
		return
	}

	if len(torrents) == 0 {
		msg := tgbotapi.NewMessage(chatID, "No torrents found")
		b.api.Send(msg)
		return
	}

	edit := tgbotapi.NewEditMessageText(chatID, messageID, "Select a torrent to manage:")
	keyboard := CreateTorrentListKeyboard(torrents, 20, page)
	edit.ReplyMarkup = &keyboard
	b.api.Send(edit)
}
