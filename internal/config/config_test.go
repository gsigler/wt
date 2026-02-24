package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot(t *testing.T) {
	t.Run("returns dir when worktree.json exists", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ConfigFile), []byte("{}"), 0644)

		got := FindProjectRoot(dir)
		if got != dir {
			t.Errorf("got %q, want %q", got, dir)
		}
	})

	t.Run("walks up directories", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ConfigFile), []byte("{}"), 0644)
		sub := filepath.Join(dir, "sub", "deep")
		os.MkdirAll(sub, 0755)

		got := FindProjectRoot(sub)
		if got != dir {
			t.Errorf("got %q, want %q", got, dir)
		}
	})

	t.Run("returns empty string at filesystem root", func(t *testing.T) {
		got := FindProjectRoot("/")
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("reads and parses config", func(t *testing.T) {
		dir := t.TempDir()
		// Resolve symlinks (macOS /var -> /private/var)
		dir, _ = filepath.EvalSymlinks(dir)
		cfg := Config{Remote: "origin", DefaultBase: "main"}
		data, _ := json.Marshal(cfg)
		os.WriteFile(filepath.Join(dir, ConfigFile), data, 0644)

		// chdir to the temp dir
		orig, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(orig)

		root, got, err := LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if root != dir {
			t.Errorf("root = %q, want %q", root, dir)
		}
		if got.Remote != "origin" {
			t.Errorf("Remote = %q, want %q", got.Remote, "origin")
		}
		if got.DefaultBase != "main" {
			t.Errorf("DefaultBase = %q, want %q", got.DefaultBase, "main")
		}
	})

	t.Run("returns error when not in a wt project", func(t *testing.T) {
		dir := t.TempDir()
		orig, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(orig)

		_, _, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestWriteConfig(t *testing.T) {
	t.Run("writes JSON with 2-space indent and trailing newline", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &Config{Remote: "origin", DefaultBase: "main"}

		if err := WriteConfig(dir, cfg); err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(filepath.Join(dir, ConfigFile))
		expected, _ := json.MarshalIndent(cfg, "", "  ")
		expected = append(expected, '\n')

		if string(data) != string(expected) {
			t.Errorf("got:\n%s\nwant:\n%s", data, expected)
		}
	})
}
