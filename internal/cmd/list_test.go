package cmd

import (
	"strings"
	"testing"

	"github.com/gradyholmes/wt/internal/config"
)

func TestListCmd(t *testing.T) {
	t.Run("calls gitInBare with worktree list and prints output", func(t *testing.T) {
		worktreeOutput := "/projects/myrepo/main  abc1234 [main]\n/projects/myrepo/feat  def5678 [feat]"

		mg := &MockGit{
			GitInBareFunc: func(args, root string) (string, error) {
				if args != "worktree list" {
					t.Errorf("unexpected git args: %s", args)
				}
				if root != "/projects/myrepo" {
					t.Errorf("unexpected root: %s", root)
				}
				return worktreeOutput, nil
			},
		}
		mc := &MockConfig{Root: "/projects/myrepo", Cfg: &config.Config{}}
		d, stdout, _ := testDeps(mg, mc, nil)

		cmd := ListCmd(d)
		cmd.SetArgs([]string{})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}

		got := strings.TrimSpace(stdout.String())
		if got != worktreeOutput {
			t.Errorf("got %q, want %q", got, worktreeOutput)
		}
	})
}
