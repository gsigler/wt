# Go Migration Plan for `wt`

## Overview

Rewrite the `wt` CLI from Node.js to Go. The tool manages git worktrees in a bare repo setup — it wraps git commands to automate cloning, worktree creation, upstream tracking, file copying, and post-create scripts. The current implementation is ~450 lines of JS across 7 modules with 41 tests.

---

## Go Library Choices

| Concern | Recommendation | Why |
|---|---|---|
| CLI framework | `cobra` | De-facto standard for Go CLIs (kubectl, gh, hugo). Subcommand routing, flag parsing, help generation. |
| Interactive prompts | `bufio.Scanner` (stdlib) | The prompts are simple single-line reads with defaults. No need for a library like `survey`/`bubbletea` — stdlib is sufficient and keeps deps minimal. |
| Git execution | `os/exec` (stdlib) | Current tool shells out to `git` via `execSync`. Same approach in Go with `exec.Command`. No need for a git library like `go-git` — shelling out keeps behavior identical. |
| Config (JSON) | `encoding/json` (stdlib) | `worktree.json` is simple JSON. Stdlib handles it directly. |
| File operations | `os`, `io`, `path/filepath` (stdlib) | Copy files, read `.git` files, walk directories. |
| Testing | `testing` (stdlib) | Standard Go test runner. Use table-driven tests. |

**Total external dependency: `cobra` only** (which pulls in `pflag`). Everything else is stdlib.

---

## Project Structure

```
wt/
├── main.go                  # Entry point, cobra root command
├── go.mod
├── go.sum
├── cmd/
│   ├── root.go              # Root command definition, version info
│   ├── init.go              # wt init [directory]
│   ├── create.go            # wt create <branch> [--base <base>]
│   ├── list.go              # wt list
│   ├── pr.go                # wt pr <number>
│   └── remove.go            # wt remove <branch> [--force]
├── internal/
│   ├── git/
│   │   └── git.go           # git() and gitInBare() helpers
│   ├── config/
│   │   └── config.go        # findProjectRoot, loadConfig, writeConfig
│   ├── worktree/
│   │   └── setup.go         # setupWorktree() shared logic
│   └── prompt/
│       └── prompt.go        # prompt(question, default) helper
├── internal/git/git_test.go
├── internal/config/config_test.go
├── internal/worktree/setup_test.go
└── cmd/
    ├── create_test.go
    ├── pr_test.go
    └── remove_test.go
```

**Key decisions:**
- `internal/` prevents external import (this is a CLI, not a library).
- `cmd/` holds cobra command definitions — each file is self-contained.
- `internal/worktree/setup.go` holds `setupWorktree()` shared by `create` and `pr` commands.
- `internal/prompt/` wraps `bufio.Scanner` with a `Prompt(question, defaultVal) string` helper, and accepts an `io.Reader` for testability.

---

## Module-by-Module Migration

### 1. `internal/git/git.go` — Git Helpers

Maps directly from `lib/git.js`.

```go
// git runs a git command and returns trimmed stdout.
func Run(args string, opts ...Option) (string, error)

// gitInBare runs a git command with GIT_DIR pointing to .bare/.
func RunInBare(args string, projectRoot string) (string, error)
```

**Details:**
- Use `exec.Command("git", strings.Fields(args)...)` instead of `execSync`.
- Set `cmd.Dir` for cwd, `cmd.Env` for `GIT_DIR`.
- Return `(string, error)` — Go convention vs. JS throwing on non-zero exit.
- For commands that need live output (fetch, post-create script), set `cmd.Stdout = os.Stdout` and `cmd.Stderr = os.Stderr`.
- Use a functional options pattern or a simple `RunOpts` struct for cwd/gitDir/stdio config.

**Gotcha:** The current JS uses string interpolation to build commands (`git ${args}`). In Go, split args properly to avoid shell injection. Use `strings.Fields()` or pass args as a slice.

### 2. `internal/config/config.go` — Config Loading

Maps directly from `lib/config.js`.

```go
type Config struct {
    Remote           string   `json:"remote"`
    DefaultBase      string   `json:"defaultBase"`
    CopyFiles        []string `json:"copyFiles"`
    CopyDirs         []string `json:"copyDirs"`
    PostCreateScript string   `json:"postCreateScript"`
}

func FindProjectRoot(startDir string) (string, error)
func Load() (root string, cfg Config, err error)
func Write(dir string, cfg Config) error
```

**Details:**
- `FindProjectRoot` walks up checking for `worktree.json` using `filepath.Dir()` in a loop.
- `Load` calls `FindProjectRoot(os.Getwd())`, reads file, unmarshals JSON.
- `Write` marshals with `json.MarshalIndent(cfg, "", "  ")` and appends `\n`.
- On failure, return errors — let callers (cobra commands) handle `os.Exit`.

### 3. `internal/prompt/prompt.go` — Interactive Prompts

Maps from the `prompt()` helpers in `init.js` and `remove.js`.

```go
func Prompt(reader io.Reader, writer io.Writer, question string, defaultVal string) string
```

**Details:**
- Accept `io.Reader`/`io.Writer` instead of hardcoding stdin/stdout for testability.
- Use `bufio.NewScanner(reader).Scan()` to read a line.
- Display default in parentheses, return default if input is empty.
- Simpler than the JS version since Go doesn't need the Promise/readline ceremony.

### 4. `internal/worktree/setup.go` — Shared Post-Creation Logic

Maps from `setupWorktree()` in `create.js`.

```go
func Setup(worktreePath string, projectRoot string, baseBranch string, cfg config.Config) error
```

**Steps (same as JS):**
1. Read `.git` file from worktree to find gitdir path.
2. Write `config.worktree` with `[core] bare = false`, `[push] default = current`, `autoSetupRemote = true`.
3. Find source worktree (base branch worktree if it exists, else project root).
4. Copy configured files with `copyFile()` — use `io.Copy` between `os.Open`/`os.Create`.
5. Copy configured directories — use `exec.Command("cp", "-crP", src, dest)` to preserve symlinks (matches current behavior).
6. Run post-create script with `exec.Command("sh", "-c", script)` with inherited stdio.

**Gotcha:** The `-crP` flags on `cp` are Linux-specific. On macOS `cp` doesn't support `-c` (reflink). Consider detecting OS or using `-rP` as fallback. Current JS has the same limitation.

### 5. `cmd/init.go` — Init Command

Maps from `lib/commands/init.js`. Most complex command (~130 lines).

```go
var initCmd = &cobra.Command{
    Use:   "init [directory]",
    Short: "Initialize a bare repo project",
    Args:  cobra.MaximumNArgs(1),
    RunE:  runInit,
}
```

**Flow (unchanged from JS):**
1. Prompt for remote URL.
2. Derive directory name from URL if not given.
3. `mkdir`, `git clone --bare <url> .bare`.
4. Write `.git` file with `gitdir: .bare`.
5. Fix bare config: add `worktreeConfig = true`, fix fetch refspec.
6. `git fetch origin`.
7. Detect default branch from `git symbolic-ref HEAD`.
8. Prompt for post-create script, copy files, copy dirs.
9. Write `worktree.json`.

### 6. `cmd/create.go` — Create Command

Maps from `lib/commands/create.js`.

```go
var createCmd = &cobra.Command{
    Use:   "create <branch>",
    Short: "Create a new worktree",
    Args:  cobra.ExactArgs(1),
    RunE:  runCreate,
}

func init() {
    createCmd.Flags().String("base", "", "base branch (defaults to config defaultBase)")
}
```

**Flow (unchanged):** fetch → check branch exists → create worktree → setupWorktree.

### 7. `cmd/list.go` — List Command

Trivial. Calls `gitInBare("worktree list", root)` and prints.

### 8. `cmd/pr.go` — PR Command

Maps from `lib/commands/pr.js`.

```go
var prCmd = &cobra.Command{
    Use:   "pr <number>",
    Short: "Create a worktree for a pull request",
    Args:  cobra.ExactArgs(1),
    RunE:  runPR,
}
```

**Flow (unchanged):** `gh pr view` → fetch → delete stale branch → create worktree under `prs/` → setupWorktree.

### 9. `cmd/remove.go` — Remove Command

Maps from `lib/commands/remove.js`.

```go
var removeCmd = &cobra.Command{
    Use:   "remove <branch>",
    Short: "Remove a worktree",
    Args:  cobra.ExactArgs(1),
    RunE:  runRemove,
}

func init() {
    removeCmd.Flags().Bool("force", false, "force removal of dirty worktree")
}
```

**Flow (unchanged):** resolve branch name → remove worktree → prompt to delete branch → use `-D` for PR worktrees, `-d` otherwise.

---

## Testing Strategy

### Approach: Same as current — mock `exec.Command`

The current JS tests mock `execSync` and `fs` operations. In Go, the equivalent approach:

1. **Make `exec.Command` injectable.** Define interfaces or function variables in `internal/git/`:

```go
// Package-level variable, overridable in tests
var ExecCommand = exec.Command
```

Tests replace `ExecCommand` with a helper that records calls and returns canned output. This is a well-established Go testing pattern (used by the Go stdlib itself in `os/exec` tests).

2. **Use `io.Reader`/`io.Writer` for prompts.** Pass `bytes.Buffer` in tests instead of stdin/stdout.

3. **Use `t.TempDir()`** for filesystem tests (config file discovery, file copying).

4. **Table-driven tests** for repetitive cases (e.g., testing multiple git command constructions).

### Test files mirror current coverage:

| JS Test | Go Test | Tests |
|---|---|---|
| `git.test.js` (8) | `internal/git/git_test.go` | Command building, trimming, GIT_DIR, stdio |
| `config.test.js` (6) | `internal/config/config_test.go` | Directory walking, JSON parsing, missing config |
| `create.test.js` (13) | `cmd/create_test.go` | Full create flow, base branches, copying, scripts |
| `list.test.js` (1) | `cmd/list_test.go` | gitInBare delegation |
| `pr.test.js` (6) | `cmd/pr_test.go` | gh CLI, stale branch, prs/ directory |
| `remove.test.js` (7) | `cmd/remove_test.go` | Removal, branch resolution, interactive delete |

---

## Migration Order

Work bottom-up from shared modules to commands, testing each layer before moving on.

### Phase 1: Scaffold
1. `go mod init` — initialize module (e.g., `github.com/<user>/wt`).
2. `main.go` + `cmd/root.go` — minimal cobra setup, `wt --help` works.
3. `go build` produces a binary.

### Phase 2: Shared Modules
4. `internal/git/git.go` + tests — port `git()` and `gitInBare()`.
5. `internal/config/config.go` + tests — port config loading/writing.
6. `internal/prompt/prompt.go` + tests — port prompt helper.

### Phase 3: Commands (simplest first)
7. `cmd/list.go` + test — trivial, validates git+config integration.
8. `cmd/create.go` + `internal/worktree/setup.go` + tests — core worktree creation.
9. `cmd/remove.go` + test — worktree removal + interactive prompt.
10. `cmd/pr.go` + test — PR worktree (builds on create).
11. `cmd/init.go` + test — most complex, depends on all other modules.

### Phase 4: Polish
12. Error messages — match current UX (helpful suggestions on failure).
13. Cross-platform check — verify `cp -crP` behavior, path separators.
14. Remove Node.js files — delete `bin/`, `lib/`, `test/`, `package.json`, `node_modules/`.
15. Update `README.md` — installation changes from `npm link` to `go install`.

---

## Behavioral Differences to Watch

| Area | Node.js (current) | Go (target) | Notes |
|---|---|---|---|
| Error handling | `process.exit(1)` scattered in commands | Return `error` from `RunE`, cobra handles exit | Cleaner separation. Commands return errors, don't call `os.Exit` directly. |
| String args | `execSync("git " + args)` (shell interpolation) | `exec.Command("git", args...)` (no shell) | Safer. But means args must be split properly — no shell globbing or pipes. |
| Sync vs async | Everything synchronous (`execSync`) | Everything synchronous (Go default) | Natural fit — Go's blocking I/O matches the current sync JS style. |
| Binary distribution | `npm install -g` / `npm link` | `go install` or prebuilt binaries via goreleaser | Simpler distribution. Single static binary, no runtime deps. |
| Directory copy | `cp -crP` via shell | Same — `exec.Command("cp", "-crP", ...)` | Keep shelling out for symlink preservation. |
| Config marshal | `JSON.stringify(cfg, null, 2) + "\n"` | `json.MarshalIndent(cfg, "", "  ")` + `"\n"` | Identical output. |
| Post-create script | `execSync(script, {cwd, stdio: "inherit"})` | `exec.Command("sh", "-c", script)` with inherited stdio | Must use `sh -c` since Go's exec doesn't invoke a shell by default. |

---

## What Gets Simpler in Go

- **No `node_modules`** — single static binary.
- **No async/Promise ceremony** — Go is synchronous by default, matching the tool's execution model.
- **No `readline` complexity** — `bufio.Scanner` is straightforward.
- **Cross-compilation** — `GOOS=darwin GOARCH=arm64 go build` for any platform.
- **Type safety** — config struct catches typos at compile time.

## What Needs Care in Go

- **Arg splitting** — `strings.Fields()` doesn't handle quoted args. If any git commands need quoted paths with spaces, use explicit arg slices instead of string splitting.
- **Shell commands** — post-create script must run through `sh -c` since Go's `exec.Command` doesn't invoke a shell.
- **Test mocking** — Go doesn't have built-in module mocking like Node's `t.mock.method()`. Use interface injection or function variables.
- **`cp -crP` portability** — reflink (`-c`) is Linux-only. Consider `os.Symlink` / manual walk for true cross-platform support, or document Linux requirement.
