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

// ResolveWorktree finds a worktree matching name using the same cascade as `wt cd`:
// exact branch, exact basename, exact relative path, then substring on branch.
// Returns the matching entry or an error describing what went wrong.
func ResolveWorktree(name string, root string, entries []WorktreeEntry) (*WorktreeEntry, error) {
	// 1. Exact branch name match
	var matches []WorktreeEntry
	for _, e := range entries {
		if e.Branch == name {
			matches = append(matches, e)
		}
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}

	// 2. Exact directory basename match
	matches = nil
	for _, e := range entries {
		if filepath.Base(e.Dir) == name {
			matches = append(matches, e)
		}
	}
	if len(matches) == 1 {
		return &matches[0], nil
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
		return &matches[0], nil
	}

	// 4. Substring match on branch name
	matches = nil
	for _, e := range entries {
		if strings.Contains(e.Branch, name) {
			matches = append(matches, e)
		}
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no worktree found matching %q", name)
	}

	lines := fmt.Sprintf("multiple worktrees match %q:", name)
	for _, m := range matches {
		lines += fmt.Sprintf("\n  %s → %s", m.Branch, m.Dir)
	}
	return nil, fmt.Errorf("%s", lines)
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

			output, err := d.Git.GitInBare("worktree list", root)
			if err != nil {
				return err
			}
			entries := ParseWorktreeList(output)

			entry, err := ResolveWorktree(args[0], root, entries)
			if err != nil {
				fmt.Fprintln(d.Stderr, err)
				return err
			}
			fmt.Fprintln(d.Stdout, entry.Dir)
			return nil
		},
	}
}
