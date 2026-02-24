package cmd

import (
	"bytes"

	"github.com/gradyholmes/wt/internal/config"
	"github.com/gradyholmes/wt/internal/git"
)

// MockGit implements GitRunner for tests.
type MockGit struct {
	GitFunc       func(args string, opts git.RunOpts) (string, error)
	GitInBareFunc func(args string, projectRoot string) (string, error)
	Calls         []string // records GitInBare args
}

func (m *MockGit) Git(args string, opts git.RunOpts) (string, error) {
	if m.GitFunc != nil {
		return m.GitFunc(args, opts)
	}
	return "", nil
}

func (m *MockGit) GitInBare(args string, projectRoot string) (string, error) {
	m.Calls = append(m.Calls, args)
	if m.GitInBareFunc != nil {
		return m.GitInBareFunc(args, projectRoot)
	}
	return "", nil
}

// MockConfig implements ConfigLoader for tests.
type MockConfig struct {
	Root string
	Cfg  *config.Config
	Err  error
}

func (m *MockConfig) LoadConfig() (string, *config.Config, error) {
	return m.Root, m.Cfg, m.Err
}

// MockPrompter implements Prompter for tests.
type MockPrompter struct {
	Answers []string
	idx     int
}

func (m *MockPrompter) Prompt(question string, defaultValue string) string {
	if m.idx < len(m.Answers) {
		answer := m.Answers[m.idx]
		m.idx++
		if answer != "" {
			return answer
		}
	}
	return defaultValue
}

// testDeps creates a Deps with mocks and captured stdout/stderr.
func testDeps(mg *MockGit, mc *MockConfig, mp *MockPrompter) (*Deps, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return &Deps{
		Git:    mg,
		Config: mc,
		Prompt: mp,
		Stdout: stdout,
		Stderr: stderr,
	}, stdout, stderr
}
