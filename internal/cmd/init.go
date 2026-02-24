package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gradyholmes/wt/internal/config"
	"github.com/gradyholmes/wt/internal/git"
	"github.com/spf13/cobra"
)

func InitCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "init [directory]",
		Short: "Clone a bare repo and configure worktree settings",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := d.Prompt.Prompt("Remote URL?", "")
			if url == "" {
				return fmt.Errorf("a remote URL is required")
			}

			directory := ""
			if len(args) > 0 {
				directory = args[0]
			}
			if directory == "" {
				// Derive directory name from URL
				directory = filepath.Base(url)
				directory = strings.TrimSuffix(directory, ".git")
			}

			targetDir, err := filepath.Abs(directory)
			if err != nil {
				return err
			}

			entries, err := os.ReadDir(targetDir)
			if err == nil && len(entries) > 0 {
				return fmt.Errorf("directory %q already exists and is not empty", directory)
			}

			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}

			// Clone bare repo into .bare
			fmt.Fprintf(d.Stdout, "\nCloning into %s/.bare ...\n", directory)
			if _, err := d.Git.Git("clone --bare "+url+" .bare", git.RunOpts{
				Cwd:    targetDir,
				Stderr: os.Stderr,
			}); err != nil {
				return fmt.Errorf("clone failed: %w", err)
			}

			// Create .git file pointing to .bare
			if err := os.WriteFile(filepath.Join(targetDir, ".git"), []byte("gitdir: .bare\n"), 0644); err != nil {
				return err
			}

			// Fix the bare repo so worktrees resolve correctly
			bareConfigPath := filepath.Join(targetDir, ".bare", "config")
			bareConfig, err := os.ReadFile(bareConfigPath)
			if err != nil {
				return err
			}
			if !strings.Contains(string(bareConfig), "worktreeConfig") {
				bareConfig = append(bareConfig, []byte("\n[extensions]\n\tworktreeConfig = true\n")...)
			}
			if err := os.WriteFile(bareConfigPath, bareConfig, 0644); err != nil {
				return err
			}

			bareDir := filepath.Join(targetDir, ".bare")

			// Fix fetch refspec
			if _, err := d.Git.Git("config remote.origin.fetch +refs/heads/*:refs/remotes/origin/*", git.RunOpts{GitDir: bareDir}); err != nil {
				return err
			}
			if _, err := d.Git.Git("fetch origin", git.RunOpts{GitDir: bareDir, Stderr: os.Stderr}); err != nil {
				return err
			}

			// Detect default branch
			defaultBranch := "main"
			headRef, err := d.Git.Git("symbolic-ref HEAD", git.RunOpts{GitDir: bareDir})
			if err == nil {
				defaultBranch = strings.TrimPrefix(headRef, "refs/heads/")
			}

			postCreateScript := d.Prompt.Prompt("Command to run after creating a worktree?", "npm install")

			copyFilesStr := d.Prompt.Prompt("Files/directories to copy into each new worktree? (comma-separated)", ".env,node_modules")
			var copyFiles []string
			for _, f := range strings.Split(copyFilesStr, ",") {
				f = strings.TrimSpace(f)
				if f != "" {
					copyFiles = append(copyFiles, f)
				}
			}

			cfg := &config.Config{
				Remote:           "origin",
				DefaultBase:      defaultBranch,
				CopyFiles:        copyFiles,
				PostCreateScript: postCreateScript,
			}

			if err := config.WriteConfig(targetDir, cfg); err != nil {
				return err
			}

			fmt.Fprintf(d.Stdout, "\nProject initialized in %s/\n", directory)
			fmt.Fprintf(d.Stdout, "  Default branch: %s\n", defaultBranch)
			if postCreateScript != "" {
				fmt.Fprintf(d.Stdout, "  Post-create script: %s\n", postCreateScript)
			} else {
				fmt.Fprintf(d.Stdout, "  Post-create script: (none)\n")
			}
			if len(copyFiles) > 0 {
				fmt.Fprintf(d.Stdout, "  Copy files: %s\n", strings.Join(copyFiles, ", "))
			} else {
				fmt.Fprintf(d.Stdout, "  Copy files: (none)\n")
			}
			fmt.Fprintf(d.Stdout, "\nNext steps:\n")
			fmt.Fprintf(d.Stdout, "  cd %s\n", directory)
			fmt.Fprintln(d.Stdout, "  wt create <branch-name>")
			return nil
		},
	}
}
