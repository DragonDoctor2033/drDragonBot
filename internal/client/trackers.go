package client

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"telegramBot/internal/models"
	"time"
)

// TrackerURLs maps tracker names to their download URL patterns
var TrackerURLs = map[string]string{
	"rutracker": "https://rutracker.org/forum/dl.php?t=",
	"kinozal":   "https://dl.kinozal.tv/download.php?id=",
}

// TorrentTrackerClient handles communication with torrent trackers
type TorrentTrackerClient struct {
	client      http.Client
	credentials map[string]models.TrackerCredentials
}

// NewTorrentTrackerClient creates a new torrent tracker client
func NewTorrentTrackerClient(credentials map[string]models.TrackerCredentials) (*TorrentTrackerClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &TorrentTrackerClient{
		client:      *client,
		credentials: credentials,
	}, nil
}

// LoginToTracker authenticates with a torrent tracker
func (t *TorrentTrackerClient) LoginToTracker(trackerName string) error {
	creds, exists := t.credentials[trackerName]
	if !exists {
		return fmt.Errorf("credentials not found for tracker: %s", trackerName)
	}

	loginURL := creds.LoginURL

	// Build form data
	formData := url.Values{}
	for key, value := range creds.FormData {
		formData.Set(key, value)
	}

	// Send login request
	resp, err := t.client.PostForm(loginURL, formData)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status code: %d", resp.StatusCode)
	}

	return nil
}

// DownloadTorrent downloads a torrent file from a tracker
func (t *TorrentTrackerClient) DownloadTorrent(trackerName, id string) ([]byte, error) {
	// Try to login to tracker
	if err := t.LoginToTracker(trackerName); err != nil {
		// If login fails, try to reconnect and login again
		if err := t.Reconnect(trackerName); err != nil {
			return nil, fmt.Errorf("reconnection failed: %w", err)
		}
	}

	// Get download URL for tracker
	baseURL, exists := TrackerURLs[trackerName]
	if !exists {
		return nil, fmt.Errorf("unknown tracker: %s", trackerName)
	}

	// Download the torrent file
	downloadURL := baseURL + id
	resp, err := t.client.Get(downloadURL)
	if err != nil {
		// If download fails, try to reconnect and try again
		if err := t.Reconnect(trackerName); err != nil {
			return nil, fmt.Errorf("reconnection failed after download error: %w", err)
		}

		// Try the download again
		resp, err = t.client.Get(downloadURL)
		if err != nil {
			return nil, fmt.Errorf("download request failed after reconnection: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If we get an unauthorized or forbidden status, try to reconnect
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			if err := t.Reconnect(trackerName); err != nil {
				return nil, fmt.Errorf("reconnection failed after unauthorized error: %w", err)
			}

			// Try the download again
			resp, err = t.client.Get(downloadURL)
			if err != nil {
				return nil, fmt.Errorf("download request failed after reconnection: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("download failed with status code: %d after reconnection", resp.StatusCode)
			}
		} else {
			return nil, fmt.Errorf("download failed with status code: %d", resp.StatusCode)
		}
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("downloaded torrent is empty")
	}

	// Validate basic torrent file structure
	if len(body) < 10 || body[0] != 'd' {
		return nil, fmt.Errorf("invalid torrent file format")
	}

	return body, nil
}

// Reconnect creates a new HTTP client and attempts to login to the specified tracker
func (t *TorrentTrackerClient) Reconnect(trackerName string) error {
	// Create a new cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Create a new client with the fresh cookies
	t.client = http.Client{
		Jar:     jar,
		Timeout: t.client.Timeout,
	}

	// Attempt to login
	return t.LoginToTracker(trackerName)
}
