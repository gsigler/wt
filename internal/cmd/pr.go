package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func PrCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "pr <number>",
		Short: "Create a worktree for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number := args[0]

			// Get PR branch name from GitHub CLI
			out, err := exec.Command("gh", "pr", "view", number, "--json", "headRefName", "-q", ".headRefName").Output()
			if err != nil {
				return fmt.Errorf("failed to get PR #%s. Is `gh` installed and authenticated?", number)
			}
			branch := strings.TrimSpace(string(out))

			root, cfg, err := d.Config.LoadConfig()
			if err != nil {
				return err
			}

			base := cfg.DefaultBase
			prsDir := filepath.Join(root, "prs")
			worktreePath := filepath.Join(prsDir, number)

			if fileOrDirExists(worktreePath) {
				return fmt.Errorf("directory \"prs/%s\" already exists", number)
			}

			// Fetch latest from remote
			fmt.Fprintf(d.Stdout, "Fetching from %s...\n", cfg.Remote)
			if _, err := d.Git.GitInBare("fetch "+cfg.Remote, root); err != nil {
				return fmt.Errorf("fetch failed: %w", err)
			}

			// Delete stale local branch if it exists
			if _, err := d.Git.GitInBare("rev-parse --verify refs/heads/"+branch, root); err == nil {
				fmt.Fprintf(d.Stdout, "Deleting stale local branch %q...\n", branch)
				if _, err := d.Git.GitInBare("branch -D "+branch, root); err != nil {
					return fmt.Errorf("failed to delete stale branch: %w", err)
				}
			}

			// Ensure prs/ directory exists
			if err := os.MkdirAll(prsDir, 0755); err != nil {
				return err
			}

			// Create worktree from the PR's remote branch
			fmt.Fprintf(d.Stdout, "Creating worktree for PR #%s (%s) from %s/%s...\n", number, branch, cfg.Remote, branch)
			gitArgs := fmt.Sprintf("worktree add %s -b %s --no-track %s/%s", worktreePath, branch, cfg.Remote, branch)
			if _, err := d.Git.GitInBare(gitArgs, root); err != nil {
				return fmt.Errorf("worktree add failed: %w", err)
			}

			if err := SetupWorktree(d, root, cfg, worktreePath, base); err != nil {
				return err
			}

			fmt.Fprintf(d.Stdout, "\nWorktree ready at ./prs/%s\n", number)
			fmt.Fprintf(d.Stdout, "  PR: #%s\n", number)
			fmt.Fprintf(d.Stdout, "  Branch: %s\n", branch)
			fmt.Fprintf(d.Stdout, "\n  cd prs/%s\n", number)
			return nil
		},
	}
}
