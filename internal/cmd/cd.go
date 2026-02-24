package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// WorktreeEntry represents a parsed line from `git worktree list`.
type WorktreeEntry struct {
	Dir    string
	Branch string
}

// ParseWorktreeList parses the output of `git worktree list` into entries.
// Skips the bare entry and blank lines.
func ParseWorktreeList(output string) []WorktreeEntry {
	re := regexp.MustCompile(`^(\S+)\s+\S+\s+\[(.+)\]$`)
	var entries []WorktreeEntry
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		dir := m[1]
		branch := m[2]
		if filepath.Base(dir) == ".bare" {
			continue
		}
		entries = append(entries, WorktreeEntry{Dir: dir, Branch: branch})
	}
	return entries
}

func CdCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "cd [name]",
		Short: "Print the path to a worktree (use with shell-init for cd)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _, err := d.Config.LoadConfig()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				fmt.Fprintln(d.Stdout, root)
				return nil
			}

			name := args[0]
			output, err := d.Git.GitInBare("worktree list", root)
			if err != nil {
				return err
			}
			entries := ParseWorktreeList(output)

			// 1. Exact branch name match
			var matches []WorktreeEntry
			for _, e := range entries {
				if e.Branch == name {
					matches = append(matches, e)
				}
			}
			if len(matches) == 1 {
				fmt.Fprintln(d.Stdout, matches[0].Dir)
				return nil
			}

			// 2. Exact directory basename match
			matches = nil
			for _, e := range entries {
				if filepath.Base(e.Dir) == name {
					matches = append(matches, e)
				}
			}
			if len(matches) == 1 {
				fmt.Fprintln(d.Stdout, matches[0].Dir)
				return nil
			}

			// 3. Exact relative path match (relative to project root)
			matches = nil
			for _, e := range entries {
				rel, err := filepath.Rel(root, e.Dir)
				if err == nil && rel == name {
					matches = append(matches, e)
				}
			}
			if len(matches) == 1 {
				fmt.Fprintln(d.Stdout, matches[0].Dir)
				return nil
			}

			// 4. Substring match on branch name
			matches = nil
			for _, e := range entries {
				if strings.Contains(e.Branch, name) {
					matches = append(matches, e)
				}
			}
			if len(matches) == 1 {
				fmt.Fprintln(d.Stdout, matches[0].Dir)
				return nil
			}

			if len(matches) == 0 {
				fmt.Fprintf(d.Stderr, "No worktree found matching %q\n", name)
				return fmt.Errorf("no worktree found matching %q", name)
			}

			fmt.Fprintf(d.Stderr, "Multiple worktrees match %q:\n", name)
			for _, m := range matches {
				fmt.Fprintf(d.Stderr, "  %s → %s\n", m.Branch, m.Dir)
			}
			return fmt.Errorf("multiple worktrees match %q", name)
		},
	}
}
