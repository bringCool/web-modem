package service

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rehiy/libgo/upgrade"
	"github.com/rehiy/web-modem/appinfo"
)

const defaultGitHubAPIBase = "https://api.github.com"

// UpdateService 自更新服务
type UpdateService struct {
	client *http.Client
}

// UpdateCheckResult 更新检查结果
type UpdateCheckResult struct {
	CurrentVersion  string              `json:"current_version"`
	LatestVersion   string              `json:"latest_version"`
	HasUpdate       bool                `json:"has_update"`
	PackageChecksum string              `json:"package_checksum,omitempty"`
	Info            *upgrade.UpdateInfo `json:"info"`
}

// GitHub API 响应类型
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Body    string        `json:"body"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func NewUpdateService() *UpdateService {
	return &UpdateService{
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

// Check 检查是否存在新版本
func (s *UpdateService) Check(ctx context.Context) (*UpdateCheckResult, error) {
	release, err := s.fetchLatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	info := &upgrade.UpdateInfo{
		Type:    "github",
		Message: "已是最新版本",
		Release: release.Body,
		Version: release.TagName,
	}

	result := &UpdateCheckResult{
		CurrentVersion: appinfo.Version,
		LatestVersion:  release.TagName,
		Info:           info,
	}

	if compareVersion(release.TagName, appinfo.Version) <= 0 {
		return result, nil
	}

	asset, ok := findRuntimeAsset(release.Assets)
	if !ok {
		return nil, fmt.Errorf("未找到当前平台的更新包: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	info.Message = "发现新版本"
	info.Package = asset.BrowserDownloadURL
	result.HasUpdate = true

	return result, nil
}

// Apply 下载并应用新版本
func (s *UpdateService) Apply(ctx context.Context) (*UpdateCheckResult, error) {
	result, err := s.Check(ctx)
	if err != nil {
		return nil, err
	}
	if !result.HasUpdate {
		return result, upgrade.ErrNoUpdate
	}

	server := newStaticUpdateServer(result.Info)
	defer server.Close()

	updater := upgrade.NewUpdater(server.URL, appinfo.Version)
	updater.Download = s.downloadPackage(ctx)
	if err := updater.Apply(); err != nil {
		return result, err
	}

	return result, nil
}

func newStaticUpdateServer(info *upgrade.UpdateInfo) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}))
}

func (s *UpdateService) fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	repo := appinfo.UpdateRepo
	if repo == "" {
		return nil, errors.New("未配置更新仓库 (UPDATE_REPO)")
	}

	reqURL := fmt.Sprintf("%s/repos/%s/releases/latest", defaultGitHubAPIBase, repo)
	resp, err := s.doRequest(ctx, reqURL, "application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("查询最新版本失败: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	if release.TagName == "" {
		return nil, errors.New("最新版本响应缺少 tag_name")
	}

	return &release, nil
}

func (s *UpdateService) downloadPackage(ctx context.Context) upgrade.DownloadHandler {
	return func(pkgURL, outputPath string) (string, error) {
		resp, err := s.doRequest(ctx, pkgURL, "application/octet-stream")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			return "", fmt.Errorf("下载更新包失败: %s %s", resp.Status, strings.TrimSpace(string(body)))
		}

		reader := io.Reader(resp.Body)

		if isGzipPackage(pkgURL, resp.Header.Get("Content-Type")) {
			gzReader, err := gzip.NewReader(reader)
			if err != nil {
				return "", err
			}
			defer gzReader.Close()
			reader = gzReader
		}

		tmpPath := outputPath + ".download"
		file, err := os.Create(tmpPath)
		if err != nil {
			return "", err
		}
		if _, err = io.Copy(file, reader); err != nil {
			file.Close()
			_ = os.Remove(tmpPath)
			return "", err
		}
		if err = file.Close(); err != nil {
			_ = os.Remove(tmpPath)
			return "", err
		}
		if err = os.Rename(tmpPath, outputPath); err != nil {
			_ = os.Remove(tmpPath)
			return "", err
		}

		return outputPath, nil
	}
}

func (s *UpdateService) doRequest(ctx context.Context, reqURL, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", appinfo.Name)
	if appinfo.UpdateToken != "" {
		req.Header.Set("Authorization", "Bearer "+appinfo.UpdateToken)
	}
	return s.client.Do(req)
}

func findRuntimeAsset(assets []githubAsset) (githubAsset, bool) {
	name := fmt.Sprintf("%s-%s-%s", appinfo.Name, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	name += ".gz"

	for _, asset := range assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func isGzipPackage(pkgURL, contentType string) bool {
	parsed, err := url.Parse(pkgURL)
	if err == nil && strings.HasSuffix(parsed.Path, ".gz") {
		return true
	}
	return strings.Contains(strings.ToLower(contentType), "gzip")
}

func compareVersion(latest, current string) int {
	if latest == current {
		return 0
	}

	latestParts := parseVersionParts(latest)
	currentParts := parseVersionParts(current)
	for i := 0; i < len(latestParts) || i < len(currentParts); i++ {
		var latestPart, currentPart int
		if i < len(latestParts) {
			latestPart = latestParts[i]
		}
		if i < len(currentParts) {
			currentPart = currentParts[i]
		}

		if latestPart > currentPart {
			return 1
		}
		if latestPart < currentPart {
			return -1
		}
	}

	return strings.Compare(latest, current)
}

func parseVersionParts(version string) []int {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	version = strings.Split(version, "-")[0]

	parts := strings.Split(version, ".")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return []int{0}
		}
		values = append(values, value)
	}
	return values
}
