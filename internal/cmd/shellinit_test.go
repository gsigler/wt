package cmd

import (
	"strings"
	"testing"
)

func TestShellInitCmd(t *testing.T) {
	t.Run("outputs a wt() function definition", func(t *testing.T) {
		d, stdout, _ := testDeps(nil, nil, nil)
		cmd := ShellInitCmd(d)
		cmd.SetArgs([]string{})
		cmd.Execute()

		if !strings.Contains(stdout.String(), "wt()") {
			t.Error("output should contain wt()")
		}
	})

	t.Run("auto-cds after create and pr", func(t *testing.T) {
		d, stdout, _ := testDeps(nil, nil, nil)
		cmd := ShellInitCmd(d)
		cmd.Execute()
		out := stdout.String()

		if !strings.Contains(out, `"create"`) {
			t.Error("should reference create")
		}
		if !strings.Contains(out, `"pr"`) {
			t.Error("should reference pr")
		}
		if !strings.Contains(out, `command wt cd "$name"`) {
			t.Error("should auto-cd after create/pr")
		}
	})

	t.Run("uses command wt to avoid recursion", func(t *testing.T) {
		d, stdout, _ := testDeps(nil, nil, nil)
		cmd := ShellInitCmd(d)
		cmd.Execute()

		if !strings.Contains(stdout.String(), "command wt") {
			t.Error("should use 'command wt' to avoid recursion")
		}
	})
}
