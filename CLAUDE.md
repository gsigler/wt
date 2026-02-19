# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

`wt` is a Node.js CLI for managing git worktrees in a bare repo setup. It wraps git commands to automate cloning, worktree creation, upstream tracking, file copying, and post-create scripts.

## Development

```sh
npm install        # install dependencies (just commander)
npm link           # symlink `wt` globally for testing
wt --help          # verify it works
```

Tests use Node's built-in test runner (`node:test`). No linter or build step exists. The CLI runs plain Node.js (no transpilation).

```sh
npm test           # run all tests
```

## Architecture

Entry point is `bin/wt.js` which uses `commander` to route to five command handlers.

**Shared modules:**
- `lib/git.js` — two helpers: `git(args, opts)` for general git calls, `gitInBare(args, projectRoot)` for commands targeting the `.bare/` directory. Both use synchronous `execSync`.
- `lib/config.js` — finds `worktree.json` by walking up from cwd, loads/writes it. `loadConfig()` returns `{ projectRoot, config }` or exits if not in a wt project.

**Commands (`lib/commands/`):**
- `init.js` — interactive setup: clones bare repo into `.bare/`, creates `.git` file pointing to it, detects default branch, writes `worktree.json`
- `create.js` — fetches, creates worktree + branch from `<remote>/<base>`, copies files, runs post-create script. Exports `setupWorktree()` for shared post-creation logic.
- `pr.js` — creates a worktree for a PR under `prs/<number>/`, using `gh` CLI to resolve the branch name
- `list.js` — thin wrapper around `git worktree list`
- `remove.js` — removes worktree, optionally deletes branch (with interactive confirmation)
- `cd.js` — resolves a worktree name to an absolute path (exact branch, basename, relative path, or substring match)
- `shell-init.js` — outputs a shell function wrapper so `wt cd` can change the parent shell's directory

**Key pattern:** All user prompts use Node's `readline` module with a local `prompt(question, default)` helper defined in `init.js` and `remove.js`.

## Config

Projects are identified by `worktree.json` at the project root. Fields: `remote`, `defaultBase`, `copyFiles` (array), `postCreateScript`.
