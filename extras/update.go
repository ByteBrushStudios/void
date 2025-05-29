package extras

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const githubRepo = "InfinityBotList/void"

type GithubRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

type UpdateCheckResponse struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	DownloadURL     string `json:"download_url"`
	ReleasePage     string `json:"release_page"`
}

func CheckLatestRelease() (*GithubRelease, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "void-maintenance-server")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func UpdateCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	info := GetVoidInfo()
	release, err := CheckLatestRelease()
	if err != nil {
		http.Error(w, "Failed to check for updates: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	updateAvailable := release.TagName != "" && release.TagName != info.Version
	resp := UpdateCheckResponse{
		CurrentVersion:  info.Version,
		LatestVersion:   release.TagName,
		UpdateAvailable: updateAvailable,
		DownloadURL:     fmt.Sprintf("https://github.com/%s/releases/latest/download/void", githubRepo),
		ReleasePage:     release.HTMLURL,
	}
	json.NewEncoder(w).Encode(resp)
}
