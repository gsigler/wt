# wt

CLI for managing git worktrees in a bare repo setup. Automates cloning, worktree creation, upstream tracking, file copying, and post-create scripts into a single command.

## Install

```sh
git clone <this-repo> && cd wt
npm install
npm link
```

## Quick Start

```sh
# Set up a new project (clones bare repo, creates config)
mkdir myproject && cd myproject
wt init

# Create a worktree
wt create feature-branch

# Check out a pull request
wt pr 123

# List worktrees
wt list

# Remove a worktree
wt remove feature-branch
```

## Project Layout

After `wt init`, your project directory looks like:

```
myproject/
├── .bare/               # Bare git repo (hidden)
├── .git                 # File pointing to .bare
├── worktree.json        # wt config
├── .env                 # Shared files copied into worktrees
├── feature-branch/      # Worktree
├── another-branch/      # Another worktree
└── prs/                 # PR review worktrees
    └── 123/             # wt pr 123
```

## Commands

### `wt init [directory]`

Interactive setup — prompts for remote URL, post-create script, and files to copy. Creates `.bare/`, `.git`, and `worktree.json`.

### `wt create <branch> [--base <base>]`

Creates a new worktree:

1. Fetches from remote
2. Creates worktree with a new branch based on `<remote>/<base>`
3. Sets upstream tracking
4. Copies configured files (e.g. `.env`)
5. Runs post-create script (e.g. `npm install`)

`--base` defaults to the `defaultBase` in `worktree.json`.

### `wt pr <number>`

Creates a worktree for a pull request, organized under `prs/`:

1. Uses `gh pr view` to get the PR's branch name
2. Fetches from remote
3. Creates worktree at `prs/<number>/`
4. Copies configured files and runs post-create script

Requires the [GitHub CLI](https://cli.github.com/) (`gh`) to be installed and authenticated.

### `wt list`

Lists all worktrees.

### `wt remove <branch> [--force]`

Removes the worktree and optionally deletes the branch. Use `--force` for dirty worktrees.

## Config

`worktree.json` lives in the project root and is created by `wt init`:

```json
{
  "remote": "origin",
  "defaultBase": "main",
  "copyFiles": [".env"],
  "postCreateScript": "npm install"
}
```

| Field | Description |
|---|---|
| `remote` | Remote name for fetching and tracking |
| `defaultBase` | Branch new worktrees are based on |
| `copyFiles` | Files from project root copied into each new worktree |
| `postCreateScript` | Command run inside the worktree after creation |
