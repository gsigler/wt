package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigFile = "worktree.json"

// Config represents the worktree.json configuration.
type Config struct {
	Remote           string   `json:"remote"`
	DefaultBase      string   `json:"defaultBase"`
	CopyFiles        []string `json:"copyFiles"`
	PostCreateScript string   `json:"postCreateScript"`
}

// FindProjectRoot walks up from startDir looking for worktree.json.
// Returns the directory containing it, or "" if not found.
func FindProjectRoot(startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, ConfigFile)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// LoadConfig finds worktree.json by walking up from cwd, reads it, and
// returns the project root and parsed config. Returns an error if not
// inside a wt project or the config can't be parsed.
func LoadConfig() (string, *Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}
	root := FindProjectRoot(cwd)
	if root == "" {
		return "", nil, fmt.Errorf("not inside a wt project. Run `wt init` first, or cd into a project directory")
	}
	data, err := os.ReadFile(filepath.Join(root, ConfigFile))
	if err != nil {
		return "", nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", nil, err
	}
	return root, &cfg, nil
}

// WriteConfig writes the config as indented JSON to dir/worktree.json.
func WriteConfig(dir string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, ConfigFile), data, 0644)
}
