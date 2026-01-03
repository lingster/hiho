package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration options.
type Config struct {
	KeyBindings KeyBindings `yaml:"keybindings"`
}

// KeyBindings defines keyboard shortcuts for the application.
type KeyBindings struct {
	Quit         string `yaml:"quit"`
	CycleWindows string `yaml:"cycle_windows"`
	NextSession  string `yaml:"next_session"`
	PrevSession  string `yaml:"prev_session"`
	ToggleTab    string `yaml:"toggle_tab"`
	SessionUp    string `yaml:"session_up"`
	SessionDown  string `yaml:"session_down"`
	FocusSidebar string `yaml:"focus_sidebar"`
	FocusMain    string `yaml:"focus_main"`
}

// DefaultConfig returns a Config with default keybindings.
func DefaultConfig() Config {
	return Config{
		KeyBindings: KeyBindings{
			Quit:         "ctrl+c",
			CycleWindows: "ctrl+o",
			NextSession:  "alt+right",
			PrevSession:  "alt+left",
			ToggleTab:    "tab",
			SessionUp:    "up",
			SessionDown:  "down",
			FocusSidebar: "ctrl+1",
			FocusMain:    "ctrl+2",
		},
	}
}

// configPath returns the path to the config file.
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "hiho", "config.yaml")
}

// LoadConfig loads configuration from the config file.
// If the file doesn't exist, it returns the default config.
func LoadConfig() Config {
	cfg := DefaultConfig()

	path := configPath()
	if path == "" {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist or can't be read, use defaults
		return cfg
	}

	// Parse YAML and merge with defaults
	var fileCfg Config
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return cfg
	}

	// Merge: only override non-empty values
	if fileCfg.KeyBindings.Quit != "" {
		cfg.KeyBindings.Quit = fileCfg.KeyBindings.Quit
	}
	if fileCfg.KeyBindings.CycleWindows != "" {
		cfg.KeyBindings.CycleWindows = fileCfg.KeyBindings.CycleWindows
	}
	if fileCfg.KeyBindings.NextSession != "" {
		cfg.KeyBindings.NextSession = fileCfg.KeyBindings.NextSession
	}
	if fileCfg.KeyBindings.PrevSession != "" {
		cfg.KeyBindings.PrevSession = fileCfg.KeyBindings.PrevSession
	}
	if fileCfg.KeyBindings.ToggleTab != "" {
		cfg.KeyBindings.ToggleTab = fileCfg.KeyBindings.ToggleTab
	}
	if fileCfg.KeyBindings.SessionUp != "" {
		cfg.KeyBindings.SessionUp = fileCfg.KeyBindings.SessionUp
	}
	if fileCfg.KeyBindings.SessionDown != "" {
		cfg.KeyBindings.SessionDown = fileCfg.KeyBindings.SessionDown
	}
	if fileCfg.KeyBindings.FocusSidebar != "" {
		cfg.KeyBindings.FocusSidebar = fileCfg.KeyBindings.FocusSidebar
	}
	if fileCfg.KeyBindings.FocusMain != "" {
		cfg.KeyBindings.FocusMain = fileCfg.KeyBindings.FocusMain
	}

	return cfg
}

// SaveDefaultConfig creates a default config file if it doesn't exist.
func SaveDefaultConfig() error {
	path := configPath()
	if path == "" {
		return nil
	}

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	// Create directory
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write default config
	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
