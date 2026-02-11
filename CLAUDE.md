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

No test framework, linter, or build step exists. The CLI runs plain Node.js (no transpilation).

## Architecture

~320 lines total. Entry point is `bin/wt.js` which uses `commander` to route to four command handlers.

**Shared modules:**
- `lib/git.js` — two helpers: `git(args, opts)` for general git calls, `gitInBare(args, projectRoot)` for commands targeting the `.bare/` directory. Both use synchronous `execSync`.
- `lib/config.js` — finds `worktree.json` by walking up from cwd, loads/writes it. `loadConfig()` returns `{ projectRoot, config }` or exits if not in a wt project.

**Commands (`lib/commands/`):**
- `init.js` — interactive setup: clones bare repo into `.bare/`, creates `.git` file pointing to it, detects default branch, writes `worktree.json`
- `create.js` — fetches, creates worktree + branch from `<remote>/<base>`, sets upstream, copies files, runs post-create script
- `list.js` — thin wrapper around `git worktree list`
- `remove.js` — removes worktree, optionally deletes branch (with interactive confirmation)

**Key pattern:** All user prompts use Node's `readline` module with a local `prompt(question, default)` helper defined in `init.js` and `remove.js`.

## Config

Projects are identified by `worktree.json` at the project root. Fields: `remote`, `defaultBase`, `copyFiles` (array), `postCreateScript`.
