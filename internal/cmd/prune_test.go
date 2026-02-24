package cmd

import (
	"strings"
	"testing"

	"github.com/gradyholmes/wt/internal/config"
)

func setupPruneTest(t *testing.T, answers []string, gitInBareImpl func(string, string) (string, error)) (*Deps, *MockGit, *MockPrompter, *strings.Builder, *strings.Builder) {
	t.Helper()
	root := t.TempDir()

	mg := &MockGit{}
	if gitInBareImpl != nil {
		mg.GitInBareFunc = gitInBareImpl
	}
	mc := &MockConfig{
		Root: root,
		Cfg: &config.Config{
			Remote:      "origin",
			DefaultBase: "main",
		},
	}
	mp := &MockPrompter{Answers: answers}
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	d := &Deps{
		Git:    mg,
		Config: mc,
		Prompt: mp,
		Stdout: stdout,
		Stderr: stderr,
	}
	return d, mg, mp, stdout, stderr
}

func TestPruneCmd(t *testing.T) {
	t.Run("identifies merged worktrees and removes them", func(t *testing.T) {
		root := ""
		d, mg, _, stdout, _ := setupPruneTest(t, []string{"y"}, func(args, projectRoot string) (string, error) {
			root = projectRoot
			if args == "worktree list" {
				return projectRoot + "/.bare           abc1234 [main]\n" +
					projectRoot + "/feat-a  def5678 [feat-a]\n" +
					projectRoot + "/feat-b  ghi9012 [feat-b]\n", nil
			}
			// feat-a is merged (merge-base succeeds), feat-b is not
			if strings.HasPrefix(args, "merge-base --is-ancestor feat-a") {
				return "", nil
			}
			if strings.HasPrefix(args, "merge-base --is-ancestor feat-b") {
				return "", &mockError{"not ancestor"}
			}
			return "", nil
		})

		cmd := PruneCmd(d)
		cmd.SilenceUsage = true
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// Should have: fetch, worktree list, 2 merge-base checks, worktree remove, branch -D
		expectedCalls := []string{
			"fetch origin",
			"worktree list",
			"merge-base --is-ancestor feat-a origin/main",
			"merge-base --is-ancestor feat-b origin/main",
			"worktree remove " + root + "/feat-a --force",
			"branch -D feat-a",
		}
		if len(mg.Calls) != len(expectedCalls) {
			t.Fatalf("expected %d git calls, got %d: %v", len(expectedCalls), len(mg.Calls), mg.Calls)
		}
		for i, want := range expectedCalls {
			if mg.Calls[i] != want {
				t.Errorf("call[%d] = %q, want %q", i, mg.Calls[i], want)
			}
		}

		if !strings.Contains(stdout.String(), "feat-a") {
			t.Errorf("stdout should mention feat-a: %s", stdout.String())
		}
	})

	t.Run("skips base branch worktree", func(t *testing.T) {
		d, mg, _, _, _ := setupPruneTest(t, nil, func(args, projectRoot string) (string, error) {
			if args == "worktree list" {
				return projectRoot + "/.bare           abc1234 [main]\n" +
					projectRoot + "/main   def5678 [main]\n", nil
			}
			return "", nil
		})

		cmd := PruneCmd(d)
		cmd.SilenceUsage = true
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// Should only have fetch + worktree list — no merge-base checks for main
		if len(mg.Calls) != 2 {
			t.Fatalf("expected 2 git calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
	})

	t.Run("dry-run prints but does not remove", func(t *testing.T) {
		d, mg, _, stdout, _ := setupPruneTest(t, nil, func(args, projectRoot string) (string, error) {
			if args == "worktree list" {
				return projectRoot + "/.bare           abc1234 [main]\n" +
					projectRoot + "/feat-a  def5678 [feat-a]\n", nil
			}
			if strings.HasPrefix(args, "merge-base --is-ancestor feat-a") {
				return "", nil
			}
			return "", nil
		})

		cmd := PruneCmd(d)
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"--dry-run"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// Should only have fetch, worktree list, merge-base — no remove or branch delete
		if len(mg.Calls) != 3 {
			t.Fatalf("expected 3 git calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if !strings.Contains(stdout.String(), "Dry run") {
			t.Errorf("stdout should mention dry run: %s", stdout.String())
		}
		if !strings.Contains(stdout.String(), "feat-a") {
			t.Errorf("stdout should list feat-a: %s", stdout.String())
		}
	})

	t.Run("no-op when nothing is merged", func(t *testing.T) {
		d, mg, _, stdout, _ := setupPruneTest(t, nil, func(args, projectRoot string) (string, error) {
			if args == "worktree list" {
				return projectRoot + "/.bare           abc1234 [main]\n" +
					projectRoot + "/feat-a  def5678 [feat-a]\n", nil
			}
			if strings.HasPrefix(args, "merge-base") {
				return "", &mockError{"not ancestor"}
			}
			return "", nil
		})

		cmd := PruneCmd(d)
		cmd.SilenceUsage = true
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// fetch + worktree list + merge-base check only
		if len(mg.Calls) != 3 {
			t.Fatalf("expected 3 git calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if !strings.Contains(stdout.String(), "No merged worktrees") {
			t.Errorf("stdout should say no merged worktrees: %s", stdout.String())
		}
	})

	t.Run("user declines confirmation", func(t *testing.T) {
		d, mg, _, stdout, _ := setupPruneTest(t, []string{"n"}, func(args, projectRoot string) (string, error) {
			if args == "worktree list" {
				return projectRoot + "/.bare           abc1234 [main]\n" +
					projectRoot + "/feat-a  def5678 [feat-a]\n", nil
			}
			if strings.HasPrefix(args, "merge-base") {
				return "", nil
			}
			return "", nil
		})

		cmd := PruneCmd(d)
		cmd.SilenceUsage = true
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		// fetch + worktree list + merge-base — no remove
		if len(mg.Calls) != 3 {
			t.Fatalf("expected 3 git calls, got %d: %v", len(mg.Calls), mg.Calls)
		}
		if !strings.Contains(stdout.String(), "Cancelled") {
			t.Errorf("stdout should say cancelled: %s", stdout.String())
		}
	})
}

// mockError implements the error interface for test stubs.
type mockError struct {
	msg string
}

func (e *mockError) Error() string { return e.msg }
