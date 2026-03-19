package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	cacheTTL       = 1 * time.Hour
	fetchTimeout   = 2 * time.Second
	npmRegistryURL = "https://registry.npmjs.org/knowns/latest"
)

type updateCache struct {
	LastChecked   int64  `json:"lastChecked"`
	LatestVersion string `json:"latestVersion"`
}

// CheckForUpdate checks npm registry for a newer version and returns
// a notification string if one is available (empty string if up to date).
// It caches the result for 1 hour at ~/.knowns/cli-cache.json.
//
// This is silent on any error — it should never interfere with normal CLI operation.
func CheckForUpdate() string {
	if shouldSkipUpdateCheck() {
		return ""
	}

	cachePath := getCachePath()
	cached := readUpdateCache(cachePath)

	var latest string
	if cached != nil && time.Since(time.UnixMilli(cached.LastChecked)) < cacheTTL {
		latest = cached.LatestVersion
	} else {
		fetched := fetchLatestVersion()
		if fetched == "" {
			return ""
		}
		latest = fetched
		writeUpdateCache(cachePath, &updateCache{
			LastChecked:   time.Now().UnixMilli(),
			LatestVersion: fetched,
		})
	}

	if latest == "" || compareVersions(latest, Version) <= 0 {
		return ""
	}

	installCmd := detectInstallCmd()
	return fmt.Sprintf("\n UPDATE  v%s available (current v%s) → %s\n", latest, Version, installCmd)
}

func shouldSkipUpdateCheck() bool {
	if os.Getenv("NO_UPDATE_CHECK") == "1" {
		return true
	}
	if os.Getenv("CI") != "" {
		return true
	}
	for _, arg := range os.Args {
		if arg == "--plain" {
			return true
		}
	}
	return false
}

func getCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".knowns", "cli-cache.json")
}

func readUpdateCache(path string) *updateCache {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var c updateCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	return &c
}

func writeUpdateCache(path string, c *updateCache) {
	if path == "" {
		return
	}
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0644)
}

func fetchLatestVersion() string {
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(npmRegistryURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.Version
}

func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var va, vb int
		if i < len(partsA) {
			va, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			vb, _ = strconv.Atoi(partsB[i])
		}
		if va > vb {
			return 1
		}
		if va < vb {
			return -1
		}
	}
	return 0
}

func detectInstallCmd() string {
	exe, _ := os.Executable()
	exePath := strings.ToLower(exe)

	// 1. Homebrew: binary in /homebrew/ or /Cellar/
	if strings.Contains(exePath, "/homebrew/") || strings.Contains(exePath, "/cellar/") {
		return "brew upgrade knowns"
	}

	// 2. Check npm_config_user_agent (set when running via npx/npm scripts)
	ua := os.Getenv("npm_config_user_agent")
	switch {
	case strings.HasPrefix(ua, "pnpm/"):
		return "pnpm add -g knowns"
	case strings.HasPrefix(ua, "yarn/"):
		return "yarn global add knowns"
	case strings.HasPrefix(ua, "bun/"):
		return "bun add -g knowns"
	case strings.HasPrefix(ua, "npm/"):
		return "npm i -g knowns"
	}

	// 3. Detect from binary path (for global installs where env var is gone)
	switch {
	case strings.Contains(exePath, "/pnpm/") || strings.Contains(exePath, "/pnpm-global/"):
		return "pnpm add -g knowns"
	case strings.Contains(exePath, "/.yarn/") || strings.Contains(exePath, "/yarn/"):
		return "yarn global add knowns"
	case strings.Contains(exePath, "/.bun/") || strings.Contains(exePath, "/bun/"):
		return "bun add -g knowns"
	case strings.Contains(exePath, "/npm/") || strings.Contains(exePath, "/node_modules/"):
		return "npm i -g knowns"
	}

	// 4. Default to npm
	return "npm i -g knowns"
}
