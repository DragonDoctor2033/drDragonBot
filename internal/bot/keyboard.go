package bot

import (
	"fmt"
	"telegramBot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CreateCategoryKeyboard creates an inline keyboard with torrent categories
func CreateCategoryKeyboard(categories map[string]models.TorrentCategory) tgbotapi.InlineKeyboardMarkup {
	row1 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Movies", "Movies."),
		tgbotapi.NewInlineKeyboardButtonData("TV Shows", "TV Shows."),
		tgbotapi.NewInlineKeyboardButtonData("Games", "Games."),
		tgbotapi.NewInlineKeyboardButtonData("Audio Books", "AudioBooks."),
	)

	row2 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Parted media", "MultiParts."),
		tgbotapi.NewInlineKeyboardButtonData("Manga", "MANGA."),
		tgbotapi.NewInlineKeyboardButtonData("Comics", "COMICS."),
	)

	return tgbotapi.NewInlineKeyboardMarkup(row1, row2)
}

// CreateTorrentActionsKeyboard creates an inline keyboard with actions for a torrent
func CreateTorrentActionsKeyboard(hash string) tgbotapi.InlineKeyboardMarkup {
	// Generate callback data with the hash
	pauseCallback := "pause:" + hash
	resumeCallback := "resume:" + hash
	deleteCallback := "delete:" + hash
	deleteWithDataCallback := "deletewithdata:" + hash
	infoCallback := "info:" + hash

	// Create keyboard rows
	row1 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⏸ Pause", pauseCallback),
		tgbotapi.NewInlineKeyboardButtonData("▶️ Resume", resumeCallback),
	)

	row2 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("ℹ️ Info", infoCallback),
	)

	row3 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🗑 Delete Torrent", deleteCallback),
		tgbotapi.NewInlineKeyboardButtonData("🗑 Delete with Files", deleteWithDataCallback),
	)

	return tgbotapi.NewInlineKeyboardMarkup(row1, row2, row3)
}

// CreateTorrentListKeyboard creates a keyboard list of torrents with pagination support
func CreateTorrentListKeyboard(torrents []models.TorrentInfo, maxButtons int, currentPage int) tgbotapi.InlineKeyboardMarkup {
	// Calculate total pages
	totalPages := (len(torrents) + maxButtons - 1) / maxButtons

	// Get torrents for current page
	startIndex := currentPage * maxButtons
	endIndex := startIndex + maxButtons
	if endIndex > len(torrents) {
		endIndex = len(torrents)
	}
	pageItems := torrents[startIndex:endIndex]

	// Create a keyboard with one torrent per row
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, torrent := range pageItems {
		// Trim name if too long
		name := torrent.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		// Create button with page information
		button := tgbotapi.NewInlineKeyboardButtonData(
			name,
			fmt.Sprintf("manage:%s:page:%d", torrent.Hash, currentPage),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	// Add pagination buttons
	var paginationRow []tgbotapi.InlineKeyboardButton
	if currentPage > 0 {
		paginationRow = append(paginationRow,
			tgbotapi.NewInlineKeyboardButtonData(
				"⬅️ Previous",
				fmt.Sprintf("list:page:%d", currentPage-1),
			),
		)
	}
	if currentPage < totalPages-1 {
		paginationRow = append(paginationRow,
			tgbotapi.NewInlineKeyboardButtonData(
				"Next ➡️",
				fmt.Sprintf("list:page:%d", currentPage+1),
			),
		)
	}

	if len(paginationRow) > 0 {
		rows = append(rows, paginationRow)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
