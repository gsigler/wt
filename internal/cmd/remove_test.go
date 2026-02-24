package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gradyholmes/wt/internal/config"
)

func setupRemoveTest(t *testing.T, answers []string, gitInBareImpl func(string, string) (string, error)) (*Deps, *MockGit, *bytes.Buffer, *bytes.Buffer, string) {
	root := t.TempDir()

	// Create fake worktree dir with .git pointer
	wtPath := filepath.Join(root, "my-branch")
	os.MkdirAll(wtPath, 0755)
	gitDir := filepath.Join(root, ".bare", "worktrees", "my-branch")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(wtPath, ".git"), []byte("gitdir: "+gitDir), 0644)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/my-branch\n"), 0644)

	mg := &MockGit{}
	if gitInBareImpl != nil {
		mg.GitInBareFunc = gitInBareImpl
	}
	mc := &MockConfig{Root: root, Cfg: &config.Config{}}
	mp := &MockPrompter{Answers: answers}
	d, stdout, stderr := testDeps(mg, mc, mp)
	return d, mg, stdout, stderr, root
}

func TestRemoveCmd(t *testing.T) {
	t.Run("removes worktree and skips branch deletion when user declines", func(t *testing.T) {
		d, mg, _, _, _ := setupRemoveTest(t, []string{"n"}, nil)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		if len(mg.Calls) != 1 {
			t.Fatalf("expected 1 git call, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if !strings.Contains(mg.Calls[0], "worktree remove") {
			t.Errorf("first call = %q", mg.Calls[0])
		}
	})

	t.Run("deletes branch when user confirms", func(t *testing.T) {
		d, mg, _, _, _ := setupRemoveTest(t, []string{"y"}, nil)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		if len(mg.Calls) != 2 {
			t.Fatalf("expected 2 git calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if mg.Calls[1] != "branch -d my-branch" {
			t.Errorf("second call = %q, want 'branch -d my-branch'", mg.Calls[1])
		}
	})

	t.Run("passes --force flag to worktree remove", func(t *testing.T) {
		d, mg, _, _, _ := setupRemoveTest(t, []string{"n"}, nil)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch", "--force"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(mg.Calls[0], "--force") {
			t.Errorf("call = %q, should contain --force", mg.Calls[0])
		}
	})

	t.Run("prompts to force when worktree is dirty", func(t *testing.T) {
		callCount := 0
		// First answer: "y" to force remove, second answer: "n" to skip branch delete
		d, mg, _, _, _ := setupRemoveTest(t, []string{"y", "n"}, func(args, root string) (string, error) {
			callCount++
			if callCount == 1 {
				return "", fmt.Errorf("modified or untracked files")
			}
			return "", nil
		})

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		if len(mg.Calls) != 2 {
			t.Fatalf("expected 2 git calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if !strings.Contains(mg.Calls[1], "--force") {
			t.Errorf("retry call = %q, should contain --force", mg.Calls[1])
		}
	})

	t.Run("cancels when user declines force remove", func(t *testing.T) {
		d, _, _, _, _ := setupRemoveTest(t, []string{"n"}, func(args, root string) (string, error) {
			return "", fmt.Errorf("modified or untracked files")
		})

		cmd := RemoveCmd(d)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "cancelled") {
			t.Errorf("error = %q, want cancelled", err)
		}
	})

	t.Run("errors on non-dirty worktree failure", func(t *testing.T) {
		d, _, _, _, _ := setupRemoveTest(t, []string{"n"}, func(args, root string) (string, error) {
			return "", fmt.Errorf("not a working tree")
		})

		cmd := RemoveCmd(d)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("handles branch deletion failure gracefully", func(t *testing.T) {
		callCount := 0
		d, _, _, stderr, _ := setupRemoveTest(t, []string{"y"}, func(args, root string) (string, error) {
			callCount++
			if callCount == 2 {
				return "", fmt.Errorf("not fully merged")
			}
			return "", nil
		})

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(stderr.String(), "not be fully merged") {
			t.Errorf("stderr = %q", stderr.String())
		}
	})

	t.Run("uses -D for PR worktrees", func(t *testing.T) {
		root := t.TempDir()

		// Create fake PR worktree
		wtPath := filepath.Join(root, "prs", "123")
		os.MkdirAll(wtPath, 0755)
		gitDir := filepath.Join(root, ".bare", "worktrees", "123")
		os.MkdirAll(gitDir, 0755)
		os.WriteFile(filepath.Join(wtPath, ".git"), []byte("gitdir: "+gitDir), 0644)
		os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/fix-bug\n"), 0644)

		mg := &MockGit{}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		mp := &MockPrompter{Answers: []string{"y"}}
		d, _, _ := testDeps(mg, mc, mp)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"prs/123"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		if len(mg.Calls) != 2 {
			t.Fatalf("expected 2 calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if mg.Calls[1] != "branch -D fix-bug" {
			t.Errorf("branch delete call = %q, want 'branch -D fix-bug'", mg.Calls[1])
		}
	})
}
