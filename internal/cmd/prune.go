package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func PruneCmd(d *Deps) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove worktrees whose branches are merged into the default base",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := d.Config.LoadConfig()
			if err != nil {
				return err
			}

			// Fetch latest from remote
			fmt.Fprintf(d.Stdout, "Fetching from %s...\n", cfg.Remote)
			if _, err := d.Git.GitInBare("fetch "+cfg.Remote, root); err != nil {
				return fmt.Errorf("fetch failed: %w", err)
			}

			// List worktrees
			output, err := d.Git.GitInBare("worktree list", root)
			if err != nil {
				return err
			}
			entries := ParseWorktreeList(output)

			// Check each worktree branch for merge status, skipping the default base
			remoteBase := cfg.Remote + "/" + cfg.DefaultBase
			var merged []WorktreeEntry
			for _, e := range entries {
				if e.Branch == cfg.DefaultBase {
					continue
				}
				_, err := d.Git.GitInBare(fmt.Sprintf("merge-base --is-ancestor %s %s", e.Branch, remoteBase), root)
				if err == nil {
					merged = append(merged, e)
				}
			}

			if len(merged) == 0 {
				fmt.Fprintln(d.Stdout, "No merged worktrees to remove.")
				return nil
			}

			// Print merged worktrees
			fmt.Fprintf(d.Stdout, "Merged worktrees (%d):\n", len(merged))
			for _, e := range merged {
				fmt.Fprintf(d.Stdout, "  %s\n", e.Branch)
			}

			if dryRun {
				fmt.Fprintln(d.Stdout, "Dry run — no changes made.")
				return nil
			}

			// Confirm
			answer := d.Prompt.Prompt("Remove these worktrees and delete their branches? (y/N)", "")
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				fmt.Fprintln(d.Stdout, "Cancelled.")
				return nil
			}

			// Remove each merged worktree and delete its branch
			for _, e := range merged {
				fmt.Fprintf(d.Stdout, "Removing %s...\n", e.Branch)
				if _, err := d.Git.GitInBare("worktree remove "+e.Dir+" --force", root); err != nil {
					fmt.Fprintf(d.Stderr, "Failed to remove worktree %s: %v\n", e.Branch, err)
					continue
				}
				if _, err := d.Git.GitInBare("branch -D "+e.Branch, root); err != nil {
					fmt.Fprintf(d.Stderr, "Failed to delete branch %s: %v\n", e.Branch, err)
				}
			}

			fmt.Fprintln(d.Stdout, "Done.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview merged worktrees without removing them")
	return cmd
}
