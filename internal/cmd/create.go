package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gradyholmes/wt/internal/config"
	"github.com/spf13/cobra"
)

// SetupWorktree configures a newly created worktree: writes config.worktree
// to disable bare mode, copies files from the base worktree, and runs the
// post-create script.
func SetupWorktree(d *Deps, root string, cfg *config.Config, worktreePath string, base string) error {
	// Read .git file to find the gitdir for this worktree
	dotGitContent, err := os.ReadFile(filepath.Join(worktreePath, ".git"))
	if err != nil {
		return fmt.Errorf("reading .git: %w", err)
	}
	wtGitDir := strings.TrimSpace(strings.TrimPrefix(string(dotGitContent), "gitdir: "))

	// Write config.worktree to disable bare mode and set push defaults
	configWorktree := "[core]\n\tbare = false\n[push]\n\tdefault = current\n\tautoSetupRemote = true\n"
	if err := os.WriteFile(filepath.Join(wtGitDir, "config.worktree"), []byte(configWorktree), 0644); err != nil {
		return fmt.Errorf("writing config.worktree: %w", err)
	}

	// Find source worktree to copy from (base branch worktree)
	sourceWorktree := filepath.Join(root, base)
	hasSource := dirExists(sourceWorktree)

	// Copy files and directories
	for _, file := range cfg.CopyFiles {
		src := filepath.Join(root, file)
		if hasSource {
			src = filepath.Join(sourceWorktree, file)
		}
		dest := filepath.Join(worktreePath, file)
		if err := copyRecursive(src, dest); err != nil {
			continue // silently skip missing files, matching Node.js behavior
		}
		fmt.Fprintln(d.Stdout, "Copied", file)
	}

	// Run post-create script
	if cfg.PostCreateScript != "" {
		fmt.Fprintln(d.Stdout, "Running:", cfg.PostCreateScript)
		c := exec.Command("sh", "-c", cfg.PostCreateScript)
		c.Dir = worktreePath
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		if err := c.Run(); err != nil {
			return fmt.Errorf("post-create script failed: %w", err)
		}
	}

	return nil
}

func CreateCmd(d *Deps) *cobra.Command {
	var base string
	cmd := &cobra.Command{
		Use:   "create <branch>",
		Short: "Create a new worktree for the given branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]

			root, cfg, err := d.Config.LoadConfig()
			if err != nil {
				return err
			}

			if base == "" {
				base = cfg.DefaultBase
			}
			worktreePath := filepath.Join(root, branch)

			if dirExists(worktreePath) {
				return fmt.Errorf("directory %q already exists", branch)
			}

			// Fetch latest from remote
			fmt.Fprintf(d.Stdout, "Fetching from %s...\n", cfg.Remote)
			if _, err := d.Git.GitInBare("fetch "+cfg.Remote, root); err != nil {
				return fmt.Errorf("fetch failed: %w", err)
			}

			// Check if branch already exists
			_, err = d.Git.GitInBare("rev-parse --verify refs/heads/"+branch, root)
			branchExists := err == nil

			// Create worktree
			fmt.Fprintf(d.Stdout, "Creating worktree for %q based on %s/%s...\n", branch, cfg.Remote, base)
			if branchExists {
				if _, err := d.Git.GitInBare(fmt.Sprintf("worktree add %s %s", worktreePath, branch), root); err != nil {
					return fmt.Errorf("worktree add failed: %w", err)
				}
			} else {
				if _, err := d.Git.GitInBare(fmt.Sprintf("worktree add %s -b %s --no-track %s/%s", worktreePath, branch, cfg.Remote, base), root); err != nil {
					return fmt.Errorf("worktree add failed: %w", err)
				}
			}

			if err := SetupWorktree(d, root, cfg, worktreePath, base); err != nil {
				return err
			}

			fmt.Fprintf(d.Stdout, "\nWorktree ready at ./%s\n", branch)
			fmt.Fprintf(d.Stdout, "  Branch: %s\n", branch)
			fmt.Fprintf(d.Stdout, "  Based on: %s/%s\n", cfg.Remote, base)
			return nil
		},
	}
	cmd.Flags().StringVar(&base, "base", "", "base branch to create from")
	return cmd
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileOrDirExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// copyRecursive copies a file or directory from src to dest.
// Returns an error if src doesn't exist.
func copyRecursive(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		// Copy single file
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		return copyFile(src, dest, info.Mode())
	}

	// Copy directory recursively
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dest, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		fi, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, fi.Mode())
	})
}

func copyFile(src, dest string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
