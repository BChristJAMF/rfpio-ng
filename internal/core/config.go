package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds RFPIO API credentials and optional endpoint overrides.
type Config struct {
	APIToken string `json:"api_token"`
	SiteURL  string `json:"site_url"`

	// Optional endpoint overrides (empty string = use default)
	UsersPath       string `json:"users_path"`
	SearchPath      string `json:"search_path"`
	AnswerGetPath   string `json:"answer_get_path"`
	AnswerAddPath   string `json:"answer_add_path"`
	AnswerDeletePath string `json:"answer_delete_path"`
	AnswerUpdatePath string `json:"answer_update_path"`
	TagsListPath    string `json:"tags_list_path"`
	TagsCreatePath  string `json:"tags_create_path"`
	TagsDeletePath  string `json:"tags_delete_path"`
	TagsUpdatePath  string `json:"tags_update_path"`
	SkillFile       string `json:"skill_file"`
}

// Default endpoint paths.
const (
	DefaultUsersPath       = "/users"
	DefaultSearchPath      = "/content-lib/search"
	DefaultAnswerGetPath   = "/answer-lib/get"
	DefaultAnswerAddPath   = "/answer-lib/add"
	DefaultAnswerDeletePath = "/answer-lib/delete"
	DefaultAnswerUpdatePath = "/answer-lib/update"
	DefaultTagsListPath    = "/content-lib/tags/list"
	DefaultTagsCreatePath  = "/content-lib/tags/create"
	DefaultTagsDeletePath  = "/content-lib/tags/delete"
	DefaultTagsUpdatePath  = "/content-lib/tags/update"
)

// LoadConfig loads RFPIO credentials.
//
// Resolution order:
//   1. RFPIO_API_TOKEN + RFPIO_SITE_URL environment variables (both must be set).
//   2. JSON file at configPath (if explicitly provided).
//   3. ~/.config/rfpio/rfpio-config.json (XDG-style).
//   4. ~/.rfpio-config.json (legacy).
//   5. ./rfpio-config.json (current directory).
func LoadConfig(configPath string) (*Config, error) {
	// Check env vars first.
	envToken := os.Getenv("RFPIO_API_TOKEN")
	envURL := os.Getenv("RFPIO_SITE_URL")
	if envToken != "" && envURL != "" {
		return &Config{
			APIToken: envToken,
			SiteURL:  envURL,
		}, nil
	}

	// If explicit path provided, use only that.
	if configPath != "" {
		p := expandTilde(configPath)
		cfg, err := readConfigFile(p)
		if err != nil {
			return nil, fmt.Errorf("configuration file not found: %s: %w", p, err)
		}
		return cfg, nil
	}

	// Search default locations in order.
	home := homeDir()
	searchPaths := []string{
		filepath.Join(home, ".config", "rfpio", "rfpio-config.json"),
		filepath.Join(home, ".rfpio-config.json"),
		filepath.Join(".", "rfpio-config.json"),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			cfg, err := readConfigFile(path)
			if err != nil {
				return nil, fmt.Errorf("invalid JSON in configuration file %s: %w", path, err)
			}
			return cfg, nil
		}
	}

	tried := strings.Join(searchPaths, ", ")
	return nil, fmt.Errorf("configuration file not found. Searched: %s", tried)
}

// readConfigFile reads and validates a JSON config file.
func readConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.APIToken == "" || cfg.SiteURL == "" {
		var missing []string
		if cfg.APIToken == "" {
			missing = append(missing, "api_token")
		}
		if cfg.SiteURL == "" {
			missing = append(missing, "site_url")
		}
		return nil, fmt.Errorf("missing required config key(s): %s", strings.Join(missing, ", "))
	}

	return &cfg, nil
}

// CreateSampleConfig creates a sample configuration file if none exists.
// Returns the path created, or "" if one already exists.
func CreateSampleConfig(configPath string) (string, error) {
	if configPath == "" {
		configPath = filepath.Join(homeDir(), ".config", "rfpio", "rfpio-config.json")
	}

	p := expandTilde(configPath)
	if _, err := os.Stat(p); err == nil {
		return "", nil // already exists
	}

	// Create parent directory.
	if dir := filepath.Dir(p); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}

	sample := map[string]string{
		"api_token": "your_api_token_here",
		"site_url":  "https://your-rfpio-instance.com",
	}

	data, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(p, data, 0o600); err != nil {
		return "", err
	}

	return p, nil
}

// expandTilde expands ~ to the user's home directory.
func expandTilde(path string) string {
	if len(path) >= 2 && path[0] == '~' && (path[1] == '/' || path[1] == '\\') {
		return filepath.Join(homeDir(), path[2:])
	}
	if path == "~" {
		return homeDir()
	}
	return path
}

// homeDir returns the user's home directory.
func homeDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		// Fallback for Windows (unlikely needed but safe).
		home = os.Getenv("USERPROFILE")
	}
	return home
}
