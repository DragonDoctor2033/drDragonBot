package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"telegramBot/internal/client"
	"telegramBot/internal/models"
	"telegramBot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// formatSize converts bytes to human-readable format
func formatSize(size int64) string {
	const (
		byte     = 1
		kilobyte = 1024 * byte
		megabyte = 1024 * kilobyte
		gigabyte = 1024 * megabyte
		terabyte = 1024 * gigabyte
	)

	switch {
	case size >= terabyte:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(terabyte))
	case size >= gigabyte:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(gigabyte))
	case size >= megabyte:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(megabyte))
	case size >= kilobyte:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(kilobyte))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// formatSpeed converts bytes/second to human-readable format
func formatSpeed(speed int64) string {
	if speed == 0 {
		return "0 B/s"
	}
	return formatSize(speed) + "/s"
}

// formatProgress formats a progress ratio as a percentage
func formatProgress(progress float64) string {
	return fmt.Sprintf("%.1f%%", progress*100)
}

// formatETA formats seconds into a human-readable time estimate
func formatETA(eta int64) string {
	if eta < 0 {
		return "∞"
	}

	hours := eta / 3600
	minutes := (eta % 3600) / 60
	seconds := eta % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// HandlePasswordCommand generates a random password
func HandlePasswordCommand(wordListPath string) (string, error) {
	password, err := utils.GeneratePassword(wordListPath)
	if err != nil {
		return "", fmt.Errorf("error generating password: %w", err)
	}
	return password, nil
}

// HandleTorrentStatus returns the status of all torrents with pagination support
func HandleTorrentStatus(qbt *client.QBittorrentClient, page int) (string, tgbotapi.InlineKeyboardMarkup, error) {
	const maxTorrentsPerPage = 10

	torrents, err := qbt.GetTorrents("")
	if err != nil {
		return "", tgbotapi.InlineKeyboardMarkup{}, fmt.Errorf("error getting torrents: %w", err)
	}

	if len(torrents) == 0 {
		return "No torrents found", tgbotapi.InlineKeyboardMarkup{}, nil
	}

	var sb strings.Builder
	sb.WriteString("📥 *Torrent Status:*\n\n")

	// Calculate pagination
	startIndex := page * maxTorrentsPerPage
	endIndex := startIndex + maxTorrentsPerPage
	if endIndex > len(torrents) {
		endIndex = len(torrents)
	}

	// Display torrents for current page
	for _, t := range torrents[startIndex:endIndex] {
		// Format the display based on torrent state
		switch t.State {
		case "downloading":
			sb.WriteString(fmt.Sprintf("🔽 *%s*\n", t.Name))
			sb.WriteString("Status: Downloading\n")
			sb.WriteString(fmt.Sprintf("Progress: %s\n", formatProgress(t.Progress)))
			sb.WriteString(fmt.Sprintf("Speed: %s\n", formatSpeed(t.Dlspeed)))
			sb.WriteString(fmt.Sprintf("ETA: %s\n", formatETA(t.Eta)))
		case "uploading", "seeding":
			sb.WriteString(fmt.Sprintf("🔼 *%s*\n", t.Name))
			sb.WriteString("Status: Seeding\n")
			sb.WriteString(fmt.Sprintf("Upload Speed: %s\n", formatSpeed(t.Upspeed)))
		case "pausedDL":
			sb.WriteString(fmt.Sprintf("⏸ *%s*\n", t.Name))
			sb.WriteString("Status: Paused\n")
			sb.WriteString(fmt.Sprintf("Progress: %s\n", formatProgress(t.Progress)))
		case "stalledDL":
			sb.WriteString(fmt.Sprintf("⚠️ *%s*\n", t.Name))
			sb.WriteString("Status: Stalled\n")
			sb.WriteString(fmt.Sprintf("Progress: %s\n", formatProgress(t.Progress)))
		case "checkingDL", "checkingUP":
			sb.WriteString(fmt.Sprintf("🔍 *%s*\n", t.Name))
			sb.WriteString("Status: Checking\n")
			sb.WriteString(fmt.Sprintf("Progress: %s\n", formatProgress(t.Progress)))
		default:
			sb.WriteString(fmt.Sprintf("📁 *%s*\n", t.Name))
			sb.WriteString(fmt.Sprintf("Status: %s\n", t.State))
		}

		sb.WriteString(fmt.Sprintf("Size: %s\n", formatSize(t.Size)))
		sb.WriteString(fmt.Sprintf("Seeds/Peers: %d/%d\n", t.NumSeeds, t.NumLeechs))
		sb.WriteString("\n")
	}

	// Add total torrents info
	sb.WriteString(fmt.Sprintf("Showing page %d of %d (Total torrents: %d)",
		page+1,
		(len(torrents)+maxTorrentsPerPage-1)/maxTorrentsPerPage,
		len(torrents)))

	// Create keyboard with pagination
	keyboard := CreateTorrentListKeyboard(torrents, maxTorrentsPerPage, page)

	return sb.String(), keyboard, nil
}

// HandleSpecificTorrentStatus returns detailed status for a specific torrent
func HandleSpecificTorrentStatus(qbt *client.QBittorrentClient, searchTerm string) (string, tgbotapi.InlineKeyboardMarkup, error) {
	if strings.HasPrefix(searchTerm, "manage:") {
		// If we receive a hash from the inline keyboard
		hash := strings.TrimPrefix(searchTerm, "manage:")
		torrent, err := qbt.GetTorrentByHash(hash)
		if err != nil {
			return "", tgbotapi.InlineKeyboardMarkup{}, err
		}
		return formatTorrentDetails(torrent), CreateTorrentActionsKeyboard(torrent.Hash), nil
	}

	// Otherwise, search by name
	torrents, err := qbt.GetTorrentsByName(searchTerm)
	if err != nil {
		return "", tgbotapi.InlineKeyboardMarkup{}, err
	}

	if len(torrents) == 0 {
		return "No matching torrents found", tgbotapi.InlineKeyboardMarkup{}, nil
	}

	// If we find exactly one torrent, show its details with action buttons
	if len(torrents) == 1 {
		return formatTorrentDetails(&torrents[0]), CreateTorrentActionsKeyboard(torrents[0].Hash), nil
	}

	// If we find multiple torrents, show a list with inline keyboard to select
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d matching torrents:\n\n", len(torrents)))
	sb.WriteString("Select a torrent to manage it:")

	return sb.String(), CreateTorrentListKeyboard(torrents, 10, 0), nil
}

// formatTorrentDetails formats detailed information about a torrent
func formatTorrentDetails(t *models.TorrentInfo) string {
	var sb strings.Builder

	// Title and basic info
	sb.WriteString(fmt.Sprintf("📥 *%s*\n\n", t.Name))

	// Status and progress
	sb.WriteString(fmt.Sprintf("Status: %s\n", t.State))
	sb.WriteString(fmt.Sprintf("Progress: %s\n", formatProgress(t.Progress)))

	// Different details based on torrent state
	if t.State == "downloading" {
		sb.WriteString(fmt.Sprintf("Download Speed: %s\n", formatSpeed(t.Dlspeed)))
		sb.WriteString(fmt.Sprintf("ETA: %s\n", formatETA(t.Eta)))
	} else if t.State == "uploading" || t.State == "seeding" {
		sb.WriteString(fmt.Sprintf("Upload Speed: %s\n", formatSpeed(t.Upspeed)))
	}

	// Size information
	sb.WriteString(fmt.Sprintf("Size: %s\n", formatSize(t.Size)))
	if t.Progress < 1.0 {
		sb.WriteString(fmt.Sprintf("Downloaded: %s\n", formatSize(t.Size-t.AmountLeft)))
		sb.WriteString(fmt.Sprintf("Remaining: %s\n", formatSize(t.AmountLeft)))
	}

	// Connection information
	sb.WriteString(fmt.Sprintf("Seeds/Peers: %d/%d\n", t.NumSeeds, t.NumLeechs))

	// Timing information
	sb.WriteString(fmt.Sprintf("Added: %s\n", time.Unix(t.AddedOn, 0).Format("2006-01-02 15:04:05")))
	if t.CompletionDate > 0 {
		sb.WriteString(fmt.Sprintf("Completed: %s\n", time.Unix(t.CompletionDate, 0).Format("2006-01-02 15:04:05")))
	}

	// Location
	sb.WriteString(fmt.Sprintf("Save Path: %s\n", t.SavePath))

	// Hash (useful for debugging)
	sb.WriteString(fmt.Sprintf("\nHash: %s", t.Hash))

	return sb.String()
}

// HandleTorrentAction performs actions on torrents (pause, resume, delete)
func HandleTorrentAction(qbt *client.QBittorrentClient, action string, hash string) (string, tgbotapi.InlineKeyboardMarkup, error) {
	// Get torrent details before taking action
	torrent, err := qbt.GetTorrentByHash(hash)
	if err != nil {
		return "", tgbotapi.InlineKeyboardMarkup{}, err
	}

	name := torrent.Name

	// Perform the requested action
	switch {
	case action == "pause":
		err = qbt.PauseTorrents([]string{hash})
		if err != nil {
			return "", tgbotapi.InlineKeyboardMarkup{}, err
		}
		return fmt.Sprintf("Paused: %s", name), CreateTorrentActionsKeyboard(hash), nil

	case action == "resume":
		err = qbt.ResumeTorrents([]string{hash})
		if err != nil {
			return "", tgbotapi.InlineKeyboardMarkup{}, err
		}
		return fmt.Sprintf("Resumed: %s", name), CreateTorrentActionsKeyboard(hash), nil

	case action == "delete":
		err = qbt.DeleteTorrents([]string{hash}, false)
		if err != nil {
			return "", tgbotapi.InlineKeyboardMarkup{}, err
		}
		return fmt.Sprintf("Deleted torrent: %s (files were kept)", name), tgbotapi.InlineKeyboardMarkup{}, nil

	case action == "deletewithdata":
		err = qbt.DeleteTorrents([]string{hash}, true)
		if err != nil {
			return "", tgbotapi.InlineKeyboardMarkup{}, err
		}
		return fmt.Sprintf("Deleted torrent and data: %s", name), tgbotapi.InlineKeyboardMarkup{}, nil

	case action == "info":
		// Refresh torrent info
		updatedTorrent, err := qbt.GetTorrentByHash(hash)
		if err != nil {
			return "", tgbotapi.InlineKeyboardMarkup{}, err
		}
		return formatTorrentDetails(updatedTorrent), CreateTorrentActionsKeyboard(hash), nil

	default:
		return "", tgbotapi.InlineKeyboardMarkup{}, fmt.Errorf("unknown action: %s", action)
	}
}

// ProcessTorrentLink extracts tracker info and ID from a torrent link
func ProcessTorrentLink(link string) (string, string, error) {
	r := regexp.MustCompile(`(http|https)://(kinozal|rutracker)\.[a-z]{2,4}\b([-a-zA-Z0-9@:%_+.~#?&/=]*)`)
	tracker := regexp.MustCompile("kinozal|rutracker")

	matches := r.FindStringSubmatch(link)
	if matches == nil {
		return "", "", fmt.Errorf("invalid torrent link format")
	}

	trackerName := tracker.FindString(link)

	// Extract ID based on tracker pattern
	var id string
	if trackerName == "rutracker" {
		idRegex := regexp.MustCompile(`t=(\d+)`)
		idMatches := idRegex.FindStringSubmatch(link)
		if len(idMatches) < 2 {
			return "", "", fmt.Errorf("could not extract ID from rutracker link")
		}
		id = idMatches[1]
	} else if trackerName == "kinozal" {
		idRegex := regexp.MustCompile(`id=(\d+)`)
		idMatches := idRegex.FindStringSubmatch(link)
		if len(idMatches) < 2 {
			return "", "", fmt.Errorf("could not extract ID from kinozal link")
		}
		id = idMatches[1]
	} else {
		return "", "", fmt.Errorf("unsupported tracker: %s", trackerName)
	}

	return trackerName, id, nil
}

// DownloadAndAddTorrent downloads a torrent from a tracker and adds it to qBittorrent
func DownloadAndAddTorrent(trackerClient *client.TorrentTrackerClient, qbtClient *client.QBittorrentClient, trackerName, id, savePath string) (string, error) {
	// Download torrent file from tracker
	torrentBytes, err := trackerClient.DownloadTorrent(trackerName, id)
	if err != nil {
		return "", fmt.Errorf("failed to download torrent: %w", err)
	}

	// Add torrent to qBittorrent
	torrent, err := qbtClient.AddTorrent(torrentBytes, savePath)
	if err != nil {
		return "", fmt.Errorf("failed to add torrent to qBittorrent: %w", err)
	}

	// Create a more detailed success message
	return fmt.Sprintf("Torrent successfully added to download queue:\n📥 *%s*\n📂 Category: %s\n💾 Save Path: %s",
		torrent.Name,
		torrent.Category,
		torrent.SavePath), nil
}
