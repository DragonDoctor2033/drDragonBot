package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"telegramBot/internal/models"
)

// QBittorrentClient handles communication with qBittorrent WebUI API
type QBittorrentClient struct {
	client     http.Client
	config     models.QBittorrentCredentials
	isLoggedIn bool
}

// NewQBittorrentClient creates a new qBittorrent client
func NewQBittorrentClient(config models.QBittorrentCredentials) (*QBittorrentClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &QBittorrentClient{
		client: *client,
		config: config,
	}, nil
}

// Login authenticates with qBittorrent WebUI
func (q *QBittorrentClient) Login() error {
	loginURL := fmt.Sprintf("%s/api/v2/auth/login", q.config.URL)
	data := url.Values{
		"username": {q.config.Username},
		"password": {q.config.Password},
	}

	resp, err := q.client.PostForm(loginURL, data)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Ok") {
		return fmt.Errorf("login failed: %s", body)
	}

	q.isLoggedIn = true
	return nil
}

// ensureLoggedIn makes sure the client is authenticated
func (q *QBittorrentClient) ensureLoggedIn() error {
	// If we think we're logged in, test the connection
	if q.isLoggedIn {
		// Make a simple API call to verify the connection
		testURL := fmt.Sprintf("%s/api/v2/app/version", q.config.URL)
		resp, err := q.client.Get(testURL)

		// If the request succeeds and returns 200 OK, we're still logged in
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}

		// If we got here, the session is likely expired
		q.isLoggedIn = false

		// Clean up if we got a response
		if err == nil {
			resp.Body.Close()
		}
	}

	// Login required
	return q.Login()
}

// AddTorrent uploads a torrent file to qBittorrent and returns the added torrent's details
func (q *QBittorrentClient) AddTorrent(torrentBytes []byte, savePath string) (*models.TorrentInfo, error) {
	if err := q.ensureLoggedIn(); err != nil {
		return nil, err
	}

	// Validate torrent file
	if len(torrentBytes) == 0 {
		return nil, fmt.Errorf("torrent file is empty")
	}

	url := fmt.Sprintf("%s/api/v2/torrents/add", q.config.URL)

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	// Create form for writing the file with .torrent extension
	formWriter, err := writer.CreateFormFile("torrents", "download.torrent")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Write torrent data
	if _, err = formWriter.Write(torrentBytes); err != nil {
		return nil, fmt.Errorf("failed to write torrent bytes: %w", err)
	}

	// Add save path
	if savePath != "" {
		if err = writer.WriteField("savepath", savePath); err != nil {
			return nil, fmt.Errorf("failed to add save path: %w", err)
		}
	}

	// Close the writer
	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", url, &buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "TelegramTorrentBot")

	// Send request
	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body for more detailed error information
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Get the list of torrents to find the newly added one
	torrents, err := q.GetTorrents("")
	if err != nil {
		return nil, fmt.Errorf("failed to get torrents after adding: %w", err)
	}

	// Find the most recently added torrent (assuming it's the one we just added)
	var newestTorrent *models.TorrentInfo
	var newestTime int64 = 0
	for _, t := range torrents {
		if t.AddedOn > newestTime {
			newestTime = t.AddedOn
			newestTorrent = &t
		}
	}

	if newestTorrent == nil {
		return nil, fmt.Errorf("could not find newly added torrent")
	}

	return newestTorrent, nil
}

// GetTorrents returns information about torrents in qBittorrent
func (q *QBittorrentClient) GetTorrents(filter string) ([]models.TorrentInfo, error) {
	if err := q.ensureLoggedIn(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v2/torrents/info", q.config.URL)
	if filter != "" {
		url += "?filter=" + filter
	}

	resp, err := q.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get torrents: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var torrents []models.TorrentInfo
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return torrents, nil
}

// PauseTorrents pauses torrents with the given hashes
func (q *QBittorrentClient) PauseTorrents(hashes []string) error {
	return q.torrentAction("pause", hashes)
}

// ResumeTorrents resumes torrents with the given hashes
func (q *QBittorrentClient) ResumeTorrents(hashes []string) error {
	return q.torrentAction("resume", hashes)
}

// DeleteTorrents deletes torrents with the given hashes
func (q *QBittorrentClient) DeleteTorrents(hashes []string, deleteFiles bool) error {
	if err := q.ensureLoggedIn(); err != nil {
		return err
	}

	link := fmt.Sprintf("%s/api/v2/torrents/delete", q.config.URL)
	data := url.Values{
		"hashes":      {strings.Join(hashes, "|")},
		"deleteFiles": {fmt.Sprintf("%t", deleteFiles)},
	}

	resp, err := q.client.PostForm(link, data)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, body)
	}

	return nil
}

// torrentAction performs actions on torrents like pause, resume
func (q *QBittorrentClient) torrentAction(action string, hashes []string) error {
	if err := q.ensureLoggedIn(); err != nil {
		return err
	}

	link := fmt.Sprintf("%s/api/v2/torrents/%s", q.config.URL, action)
	data := url.Values{
		"hashes": {strings.Join(hashes, "|")},
	}

	resp, err := q.client.PostForm(link, data)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", action, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s failed with status %d: %s", action, resp.StatusCode, body)
	}

	return nil
}

// GetTorrentsByName searches for torrents with a name containing searchTerm
func (q *QBittorrentClient) GetTorrentsByName(searchTerm string) ([]models.TorrentInfo, error) {
	torrents, err := q.GetTorrents("")
	if err != nil {
		return nil, err
	}

	searchTerm = strings.ToLower(searchTerm)
	var result []models.TorrentInfo

	for _, t := range torrents {
		if strings.Contains(strings.ToLower(t.Name), searchTerm) {
			result = append(result, t)
		}
	}

	return result, nil
}

// GetTorrentByHash gets a specific torrent by its hash
func (q *QBittorrentClient) GetTorrentByHash(hash string) (*models.TorrentInfo, error) {
	torrents, err := q.GetTorrents("")
	if err != nil {
		return nil, err
	}

	hash = strings.ToLower(hash)
	for _, t := range torrents {
		if strings.EqualFold(t.Hash, hash) {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("torrent with hash %s not found", hash)
}

// Reconnect forces a new connection to qBittorrent
func (q *QBittorrentClient) Reconnect() error {
	// Reset the client's jar to clear cookies
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Create a new client with the same timeout but fresh cookies
	q.client = http.Client{
		Jar:     jar,
		Timeout: q.client.Timeout,
	}

	// Reset login status
	q.isLoggedIn = false

	// Attempt to login
	return q.Login()
}
