package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

func RemoveCmd(d *Deps) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <branch>",
		Short: "Remove a worktree and optionally delete the branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]

			root, _, err := d.Config.LoadConfig()
			if err != nil {
				return err
			}
			worktreePath := filepath.Join(root, branch)

			// Resolve the actual git branch name from the worktree before removing it
			branchName := branch
			dotGitContent, err := os.ReadFile(filepath.Join(worktreePath, ".git"))
			if err == nil {
				wtGitDir := strings.TrimSpace(strings.TrimPrefix(string(dotGitContent), "gitdir: "))
				headContent, err := os.ReadFile(filepath.Join(wtGitDir, "HEAD"))
				if err == nil {
					re := regexp.MustCompile(`^ref: refs/heads/(.+)$`)
					if m := re.FindStringSubmatch(strings.TrimSpace(string(headContent))); m != nil {
						branchName = m[1]
					}
				}
			}

			// Remove worktree
			fmt.Fprintf(d.Stdout, "Removing worktree %q...\n", branch)
			removeArgs := "worktree remove " + worktreePath
			if force {
				removeArgs += " --force"
			}
			if _, err := d.Git.GitInBare(removeArgs, root); err != nil {
				// If not already forcing and the error is about dirty files, prompt to force
				if !force && strings.Contains(err.Error(), "modified or untracked") {
					answer := d.Prompt.Prompt("Worktree has modified or untracked files. Force remove? (y/N)", "")
					if strings.ToLower(strings.TrimSpace(answer)) == "y" {
						if _, err := d.Git.GitInBare("worktree remove "+worktreePath+" --force", root); err != nil {
							return fmt.Errorf("force removal failed: %w", err)
						}
					} else {
						return fmt.Errorf("worktree removal cancelled")
					}
				} else {
					return fmt.Errorf("failed to remove worktree: %w", err)
				}
			}

			// Ask whether to delete the branch
			answer := d.Prompt.Prompt(fmt.Sprintf("Delete branch %q as well? (y/N)", branchName), "")

			if strings.ToLower(strings.TrimSpace(answer)) == "y" {
				// Use -D (force) for PR worktrees since the branch will be recreated
				// from remote by `wt pr`. Use -d (safe) for regular worktrees.
				isPr := strings.HasPrefix(branch, "prs/") || strings.HasPrefix(branch, "prs\\")
				deleteFlag := "-d"
				if isPr {
					deleteFlag = "-D"
				}
				if _, err := d.Git.GitInBare(fmt.Sprintf("branch %s %s", deleteFlag, branchName), root); err != nil {
					fmt.Fprintf(d.Stderr, "Could not delete branch %q. It may not be fully merged.\n", branchName)
					fmt.Fprintf(d.Stderr, "Use `git branch -D %s` to force delete.\n", branchName)
				} else {
					fmt.Fprintf(d.Stdout, "Branch %q deleted.\n", branchName)
				}
			}

			fmt.Fprintln(d.Stdout, "Done.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "force removal even if worktree is dirty")
	return cmd
}
