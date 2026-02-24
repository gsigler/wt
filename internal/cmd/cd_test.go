package cmd

import (
	"strings"
	"testing"

	"github.com/gradyholmes/wt/internal/config"
)

const testWorktreeOutput = `/projects/myrepo/.bare           (bare)
/projects/myrepo/main            abc1234 [main]
/projects/myrepo/feature-branch  def5678 [feature-branch]
/projects/myrepo/prs/123         ghi9012 [fix/login-bug]
/projects/myrepo/prs/456         jkl3456 [feat/signup]`

func cdDeps() (*MockGit, *MockConfig) {
	mg := &MockGit{
		GitInBareFunc: func(args, root string) (string, error) {
			return testWorktreeOutput, nil
		},
	}
	mc := &MockConfig{Root: "/projects/myrepo", Cfg: &config.Config{}}
	return mg, mc
}

func TestParseWorktreeList(t *testing.T) {
	entries := ParseWorktreeList(testWorktreeOutput)
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}
	// .bare should be excluded
	for _, e := range entries {
		if strings.Contains(e.Dir, ".bare") {
			t.Error("should exclude .bare entry")
		}
	}
}

func TestCdCmd(t *testing.T) {
	t.Run("prints project root when no name given", func(t *testing.T) {
		mg, mc := cdDeps()
		d, stdout, _ := testDeps(mg, mc, nil)

		cmd := CdCmd(d)
		cmd.SetArgs([]string{})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(stdout.String()); got != "/projects/myrepo" {
			t.Errorf("got %q, want /projects/myrepo", got)
		}
	})

	t.Run("matches exact branch name", func(t *testing.T) {
		mg, mc := cdDeps()
		d, stdout, _ := testDeps(mg, mc, nil)

		cmd := CdCmd(d)
		cmd.SetArgs([]string{"feature-branch"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(stdout.String()); got != "/projects/myrepo/feature-branch" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("matches directory basename for PR worktrees", func(t *testing.T) {
		mg, mc := cdDeps()
		d, stdout, _ := testDeps(mg, mc, nil)

		cmd := CdCmd(d)
		cmd.SetArgs([]string{"123"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(stdout.String()); got != "/projects/myrepo/prs/123" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("matches relative path", func(t *testing.T) {
		mg, mc := cdDeps()
		d, stdout, _ := testDeps(mg, mc, nil)

		cmd := CdCmd(d)
		cmd.SetArgs([]string{"prs/123"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(stdout.String()); got != "/projects/myrepo/prs/123" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("matches substring on branch name", func(t *testing.T) {
		mg, mc := cdDeps()
		d, stdout, _ := testDeps(mg, mc, nil)

		cmd := CdCmd(d)
		cmd.SetArgs([]string{"login"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(stdout.String()); got != "/projects/myrepo/prs/123" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("errors when no match found", func(t *testing.T) {
		mg, mc := cdDeps()
		d, _, stderr := testDeps(mg, mc, nil)

		cmd := CdCmd(d)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"nonexistent"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(stderr.String(), `No worktree found matching "nonexistent"`) {
			t.Errorf("stderr = %q", stderr.String())
		}
	})

	t.Run("errors when multiple matches found", func(t *testing.T) {
		mg, mc := cdDeps()
		d, _, stderr := testDeps(mg, mc, nil)

		cmd := CdCmd(d)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"feat"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(stderr.String(), `Multiple worktrees match "feat"`) {
			t.Errorf("stderr = %q", stderr.String())
		}
	})

	t.Run("excludes .bare entry", func(t *testing.T) {
		mg, mc := cdDeps()
		d, _, stderr := testDeps(mg, mc, nil)

		cmd := CdCmd(d)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{".bare"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(stderr.String(), "No worktree found") {
			t.Errorf("stderr = %q", stderr.String())
		}
	})
}
