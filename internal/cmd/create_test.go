package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gradyholmes/wt/internal/config"
)

func TestCreateCmd(t *testing.T) {
	t.Run("fetches, creates worktree, and sets upstream", func(t *testing.T) {
		root := t.TempDir()
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if strings.Contains(args, "rev-parse") {
					return "", fmt.Errorf("not found")
				}
				// For worktree add, create the worktree dir and .git file
				if strings.Contains(args, "worktree add") {
					wtPath := filepath.Join(root, "my-branch")
					os.MkdirAll(wtPath, 0755)
					gitDir := filepath.Join(root, ".bare", "worktrees", "my-branch")
					os.MkdirAll(gitDir, 0755)
					os.WriteFile(filepath.Join(wtPath, ".git"), []byte("gitdir: "+gitDir), 0644)
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{
			Remote:      "origin",
			DefaultBase: "main",
		}}
		d, _, _ := testDeps(mg, mc, nil)

		cmd := CreateCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		calls := mg.Calls
		if len(calls) < 3 {
			t.Fatalf("expected at least 3 git calls, got %d: %v", len(calls), calls)
		}
		if calls[0] != "fetch origin" {
			t.Errorf("first call = %q, want fetch origin", calls[0])
		}
		addCall := calls[2]
		if !strings.Contains(addCall, "worktree add") {
			t.Errorf("third call should be worktree add, got %q", addCall)
		}
		if !strings.Contains(addCall, "-b my-branch") {
			t.Errorf("should create branch my-branch, got %q", addCall)
		}
		if !strings.Contains(addCall, "origin/main") {
			t.Errorf("should be based on origin/main, got %q", addCall)
		}
	})

	t.Run("uses custom base branch from --base flag", func(t *testing.T) {
		root := t.TempDir()
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if strings.Contains(args, "rev-parse") {
					return "", fmt.Errorf("not found")
				}
				if strings.Contains(args, "worktree add") {
					wtPath := filepath.Join(root, "my-branch")
					os.MkdirAll(wtPath, 0755)
					gitDir := filepath.Join(root, ".bare", "worktrees", "my-branch")
					os.MkdirAll(gitDir, 0755)
					os.WriteFile(filepath.Join(wtPath, ".git"), []byte("gitdir: "+gitDir), 0644)
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{
			Remote:      "origin",
			DefaultBase: "main",
		}}
		d, _, _ := testDeps(mg, mc, nil)

		cmd := CreateCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch", "--base", "develop"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		addCall := mg.Calls[2]
		if !strings.Contains(addCall, "origin/develop") {
			t.Errorf("should use develop base, got %q", addCall)
		}
	})

	t.Run("errors if branch directory already exists", func(t *testing.T) {
		root := t.TempDir()
		os.MkdirAll(filepath.Join(root, "my-branch"), 0755)

		mg := &MockGit{}
		mc := &MockConfig{Root: root, Cfg: &config.Config{
			Remote:      "origin",
			DefaultBase: "main",
		}}
		d, _, _ := testDeps(mg, mc, nil)

		cmd := CreateCmd(d)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error = %q", err)
		}
	})

	t.Run("writes worktree config to disable bare mode", func(t *testing.T) {
		root := t.TempDir()
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if strings.Contains(args, "rev-parse") {
					return "", fmt.Errorf("not found")
				}
				if strings.Contains(args, "worktree add") {
					wtPath := filepath.Join(root, "my-branch")
					os.MkdirAll(wtPath, 0755)
					gitDir := filepath.Join(root, ".bare", "worktrees", "my-branch")
					os.MkdirAll(gitDir, 0755)
					os.WriteFile(filepath.Join(wtPath, ".git"), []byte("gitdir: "+gitDir), 0644)
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{
			Remote:      "origin",
			DefaultBase: "main",
		}}
		d, _, _ := testDeps(mg, mc, nil)

		cmd := CreateCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		configWorktree := filepath.Join(root, ".bare", "worktrees", "my-branch", "config.worktree")
		data, err := os.ReadFile(configWorktree)
		if err != nil {
			t.Fatal("config.worktree should be written")
		}
		if !strings.Contains(string(data), "bare = false") {
			t.Error("should contain bare = false")
		}
		if !strings.Contains(string(data), "autoSetupRemote = true") {
			t.Error("should contain autoSetupRemote = true")
		}
	})

	t.Run("copies files that exist at source", func(t *testing.T) {
		root := t.TempDir()
		// Create base worktree with files to copy
		os.MkdirAll(filepath.Join(root, "main"), 0755)
		os.WriteFile(filepath.Join(root, "main", ".env"), []byte("SECRET=1"), 0644)
		os.WriteFile(filepath.Join(root, "main", "config.json"), []byte("{}"), 0644)

		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if strings.Contains(args, "rev-parse") {
					return "", fmt.Errorf("not found")
				}
				if strings.Contains(args, "worktree add") {
					wtPath := filepath.Join(root, "my-branch")
					os.MkdirAll(wtPath, 0755)
					gitDir := filepath.Join(root, ".bare", "worktrees", "my-branch")
					os.MkdirAll(gitDir, 0755)
					os.WriteFile(filepath.Join(wtPath, ".git"), []byte("gitdir: "+gitDir), 0644)
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{
			Remote:      "origin",
			DefaultBase: "main",
			CopyFiles:   []string{".env", "config.json"},
		}}
		d, stdout, _ := testDeps(mg, mc, nil)

		cmd := CreateCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// Verify files were copied
		data, err := os.ReadFile(filepath.Join(root, "my-branch", ".env"))
		if err != nil {
			t.Fatal("should copy .env")
		}
		if string(data) != "SECRET=1" {
			t.Errorf(".env content = %q", data)
		}

		if !strings.Contains(stdout.String(), "Copied .env") {
			t.Error("should log copied files")
		}
	})

	t.Run("skips missing copy files silently", func(t *testing.T) {
		root := t.TempDir()
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if strings.Contains(args, "rev-parse") {
					return "", fmt.Errorf("not found")
				}
				if strings.Contains(args, "worktree add") {
					wtPath := filepath.Join(root, "my-branch")
					os.MkdirAll(wtPath, 0755)
					gitDir := filepath.Join(root, ".bare", "worktrees", "my-branch")
					os.MkdirAll(gitDir, 0755)
					os.WriteFile(filepath.Join(wtPath, ".git"), []byte("gitdir: "+gitDir), 0644)
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{
			Remote:      "origin",
			DefaultBase: "main",
			CopyFiles:   []string{".env"},
		}}
		d, stdout, _ := testDeps(mg, mc, nil)

		cmd := CreateCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		if strings.Contains(stdout.String(), "Copied") {
			t.Error("should not log copies for missing files")
		}
	})
}
