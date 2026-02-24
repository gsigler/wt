package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gradyholmes/wt/internal/config"
)

func worktreeListOutput(root string, entries ...string) string {
	// entries are like "my-branch abc1234 [my-branch]"
	var lines []string
	lines = append(lines, root+"/.bare           abc1234 (bare)")
	for _, e := range entries {
		lines = append(lines, e)
	}
	return strings.Join(lines, "\n")
}

func TestRemoveCmd(t *testing.T) {
	t.Run("removes worktree and skips branch deletion when user declines", func(t *testing.T) {
		root := t.TempDir()
		wtPath := filepath.Join(root, "my-branch")
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if args == "worktree list" {
					return worktreeListOutput(root, wtPath+" def5678 [my-branch]"), nil
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		mp := &MockPrompter{Answers: []string{"n"}}
		d, _, _ := testDeps(mg, mc, mp)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// worktree list + worktree remove
		if len(mg.Calls) != 2 {
			t.Fatalf("expected 2 git calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if !strings.Contains(mg.Calls[1], "worktree remove") {
			t.Errorf("second call = %q, want worktree remove", mg.Calls[1])
		}
	})

	t.Run("deletes branch when user confirms", func(t *testing.T) {
		root := t.TempDir()
		wtPath := filepath.Join(root, "my-branch")
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if args == "worktree list" {
					return worktreeListOutput(root, wtPath+" def5678 [my-branch]"), nil
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		mp := &MockPrompter{Answers: []string{"y"}}
		d, _, _ := testDeps(mg, mc, mp)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// worktree list + worktree remove + branch -d
		if len(mg.Calls) != 3 {
			t.Fatalf("expected 3 git calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if mg.Calls[2] != "branch -d my-branch" {
			t.Errorf("third call = %q, want 'branch -d my-branch'", mg.Calls[2])
		}
	})

	t.Run("passes --force flag to worktree remove", func(t *testing.T) {
		root := t.TempDir()
		wtPath := filepath.Join(root, "my-branch")
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if args == "worktree list" {
					return worktreeListOutput(root, wtPath+" def5678 [my-branch]"), nil
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		mp := &MockPrompter{Answers: []string{"n"}}
		d, _, _ := testDeps(mg, mc, mp)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch", "--force"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(mg.Calls[1], "--force") {
			t.Errorf("call = %q, should contain --force", mg.Calls[1])
		}
	})

	t.Run("prompts to force when worktree is dirty", func(t *testing.T) {
		root := t.TempDir()
		wtPath := filepath.Join(root, "my-branch")
		removeCallCount := 0
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if args == "worktree list" {
					return worktreeListOutput(root, wtPath+" def5678 [my-branch]"), nil
				}
				if strings.HasPrefix(args, "worktree remove") {
					removeCallCount++
					if removeCallCount == 1 {
						return "", fmt.Errorf("modified or untracked files")
					}
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		// First answer: "y" to force remove, second answer: "n" to skip branch delete
		mp := &MockPrompter{Answers: []string{"y", "n"}}
		d, _, _ := testDeps(mg, mc, mp)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"my-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// worktree list + worktree remove (fail) + worktree remove --force
		if len(mg.Calls) != 3 {
			t.Fatalf("expected 3 git calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if !strings.Contains(mg.Calls[2], "--force") {
			t.Errorf("retry call = %q, should contain --force", mg.Calls[2])
		}
	})

	t.Run("cancels when user declines force remove", func(t *testing.T) {
		root := t.TempDir()
		wtPath := filepath.Join(root, "my-branch")
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if args == "worktree list" {
					return worktreeListOutput(root, wtPath+" def5678 [my-branch]"), nil
				}
				if strings.HasPrefix(args, "worktree remove") {
					return "", fmt.Errorf("modified or untracked files")
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		mp := &MockPrompter{Answers: []string{"n"}}
		d, _, _ := testDeps(mg, mc, mp)

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
		root := t.TempDir()
		wtPath := filepath.Join(root, "my-branch")
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if args == "worktree list" {
					return worktreeListOutput(root, wtPath+" def5678 [my-branch]"), nil
				}
				if strings.HasPrefix(args, "worktree remove") {
					return "", fmt.Errorf("not a working tree")
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		mp := &MockPrompter{Answers: []string{"n"}}
		d, _, _ := testDeps(mg, mc, mp)

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
		root := t.TempDir()
		wtPath := filepath.Join(root, "my-branch")
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if args == "worktree list" {
					return worktreeListOutput(root, wtPath+" def5678 [my-branch]"), nil
				}
				if strings.HasPrefix(args, "branch") {
					return "", fmt.Errorf("not fully merged")
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		mp := &MockPrompter{Answers: []string{"y"}}
		d, _, stderr := testDeps(mg, mc, mp)

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
		prPath := filepath.Join(root, "prs", "123")
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if args == "worktree list" {
					return worktreeListOutput(root, prPath+" ghi9012 [fix-bug]"), nil
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		mp := &MockPrompter{Answers: []string{"y"}}
		d, _, _ := testDeps(mg, mc, mp)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"fix-bug"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// worktree list + worktree remove + branch -D
		if len(mg.Calls) != 3 {
			t.Fatalf("expected 3 calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if mg.Calls[2] != "branch -D fix-bug" {
			t.Errorf("branch delete call = %q, want 'branch -D fix-bug'", mg.Calls[2])
		}
	})

	t.Run("resolves PR worktree by branch name", func(t *testing.T) {
		root := t.TempDir()
		prPath := filepath.Join(root, "prs", "2055")
		mg := &MockGit{
			GitInBareFunc: func(args, projectRoot string) (string, error) {
				if args == "worktree list" {
					return worktreeListOutput(root,
						filepath.Join(root, "main")+" abc1234 [main]",
						prPath+" def5678 [feat/se-1284-pt-1]",
					), nil
				}
				return "", nil
			},
		}
		mc := &MockConfig{Root: root, Cfg: &config.Config{}}
		mp := &MockPrompter{Answers: []string{"n"}}
		d, stdout, _ := testDeps(mg, mc, mp)

		cmd := RemoveCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"feat/se-1284-pt-1"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// Should resolve correctly and remove the right path
		if !strings.Contains(mg.Calls[1], prPath) {
			t.Errorf("worktree remove call = %q, want path containing %s", mg.Calls[1], prPath)
		}
		if !strings.Contains(stdout.String(), "feat/se-1284-pt-1") {
			t.Errorf("stdout = %q, should mention branch name", stdout.String())
		}
	})
}
