package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gradyholmes/wt/internal/config"
	"github.com/gradyholmes/wt/internal/git"
)

// GitRunner abstracts git operations for testability.
type GitRunner interface {
	Git(args string, opts git.RunOpts) (string, error)
	GitInBare(args string, projectRoot string) (string, error)
}

// ConfigLoader abstracts config operations for testability.
type ConfigLoader interface {
	LoadConfig() (string, *config.Config, error)
}

// Prompter reads interactive input from the user.
type Prompter interface {
	Prompt(question string, defaultValue string) string
}

// Deps holds injectable dependencies for all commands.
type Deps struct {
	Git    GitRunner
	Config ConfigLoader
	Prompt Prompter
	Stdout io.Writer
	Stderr io.Writer
	Getwd  func() (string, error)
}

// DefaultDeps returns Deps wired to real implementations.
func DefaultDeps() *Deps {
	return &Deps{
		Git:    &RealGit{},
		Config: &RealConfig{},
		Prompt: &RealPrompter{Scanner: bufio.NewScanner(os.Stdin), Out: os.Stdout},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getwd:  os.Getwd,
	}
}

// RealGit implements GitRunner using the real git package.
type RealGit struct{}

func (r *RealGit) Git(args string, opts git.RunOpts) (string, error) {
	return git.Git(args, opts)
}

func (r *RealGit) GitInBare(args string, projectRoot string) (string, error) {
	return git.GitInBare(args, projectRoot)
}

// RealConfig implements ConfigLoader using the real config package.
type RealConfig struct{}

func (r *RealConfig) LoadConfig() (string, *config.Config, error) {
	return config.LoadConfig()
}

// RealPrompter implements Prompter using bufio.Scanner.
type RealPrompter struct {
	Scanner *bufio.Scanner
	Out     io.Writer
}

func (p *RealPrompter) Prompt(question string, defaultValue string) string {
	suffix := ""
	if defaultValue != "" {
		suffix = fmt.Sprintf(" (%s)", defaultValue)
	}
	fmt.Fprintf(p.Out, "%s%s: ", question, suffix)
	if p.Scanner.Scan() {
		answer := strings.TrimSpace(p.Scanner.Text())
		if answer != "" {
			return answer
		}
	}
	return defaultValue
}
