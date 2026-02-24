package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunOpts configures how a git command is executed.
type RunOpts struct {
	Cwd    string
	GitDir string
	Stderr *os.File // nil means pipe (suppress), set to os.Stderr to inherit
}

// Git runs a git command with the given args string and options.
// Args are split on whitespace (no quoted args needed in practice).
// Returns trimmed stdout.
func Git(args string, opts RunOpts) (string, error) {
	parts := append([]string{}, strings.Fields(args)...)
	c := exec.Command("git", parts...)
	if opts.Cwd != "" {
		c.Dir = opts.Cwd
	}
	if opts.GitDir != "" {
		c.Env = append(os.Environ(), "GIT_DIR="+opts.GitDir)
	}
	if opts.Stderr != nil {
		c.Stderr = opts.Stderr
	}
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GitInBare runs a git command with GIT_DIR set to <projectRoot>/.bare.
func GitInBare(args string, projectRoot string) (string, error) {
	return Git(args, RunOpts{GitDir: filepath.Join(projectRoot, ".bare")})
}
