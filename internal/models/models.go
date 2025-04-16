package models

// TorrentInfo represents information about a torrent from qBittorrent
type TorrentInfo struct {
	Name            string  `json:"name"`
	Hash            string  `json:"hash"`
	Size            int64   `json:"size"`
	Progress        float64 `json:"progress"`
	Dlspeed         int64   `json:"dlspeed"`
	Upspeed         int64   `json:"upspeed"`
	State           string  `json:"state"`
	NumSeeds        int     `json:"num_seeds"`
	NumLeechs       int     `json:"num_leechs"`
	TimeElapsed     int64   `json:"time_elapsed"`
	Eta             int64   `json:"eta"`
	SavePath        string  `json:"save_path"`
	CompletionOn    int64   `json:"completion_on"`
	RatioLimit      float64 `json:"ratio_limit"`
	SeqDl           bool    `json:"seq_dl"`
	ForceStart      bool    `json:"force_start"`
	SuperSeeding    bool    `json:"super_seeding"`
	ContentPath     string  `json:"content_path"`
	AddedOn         int64   `json:"added_on"`
	AmountLeft      int64   `json:"amount_left"`
	Category        string  `json:"category"`
	Tags            string  `json:"tags"`
	CompletionDate  int64   `json:"completion_date"`
	DownloadLimit   int64   `json:"dl_limit"`
	UploadLimit     int64   `json:"up_limit"`
	DownloadedTotal int64   `json:"downloaded"`
	UploadedTotal   int64   `json:"uploaded"`
	Ratio           float64 `json:"ratio"`
}

// TorrentCategory represents a download category and its corresponding save path
type TorrentCategory struct {
	Name     string
	SavePath string
	Callback string
}

// TrackerCredentials contains authentication information for torrent trackers
type TrackerCredentials struct {
	LoginURL string
	Username string
	Password string
	LoginKey string
	FormData map[string]string
}

// QBittorrentCredentials contains authentication information for qBittorrent
type QBittorrentCredentials struct {
	URL      string
	Username string
	Password string
}
