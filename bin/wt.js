#!/usr/bin/env node

const { program } = require("commander");
const init = require("../lib/commands/init");
const create = require("../lib/commands/create");
const list = require("../lib/commands/list");
const remove = require("../lib/commands/remove");
const pr = require("../lib/commands/pr");
const cd = require("../lib/commands/cd");
const shellInit = require("../lib/commands/shell-init");

program.name("wt").description("Git worktree CLI for bare repo workflows");

program
  .command("init")
  .description("Clone a bare repo and configure worktree settings")
  .argument("[directory]", "directory name (defaults to repo name)")
  .action(init);

program
  .command("create <branch>")
  .description("Create a new worktree for the given branch")
  .option("--base <base>", "base branch to create from")
  .action(create);

program
  .command("list")
  .description("List all worktrees")
  .action(list);

program
  .command("pr <number>")
  .description("Create a worktree for a pull request")
  .action(pr);

program
  .command("remove <branch>")
  .description("Remove a worktree and optionally delete the branch")
  .option("--force", "force removal even if worktree is dirty")
  .action(remove);

program
  .command("cd [name]")
  .description("Print the path to a worktree (use with shell-init for cd)")
  .action(cd);

program
  .command("shell-init")
  .description("Output shell function for wt cd integration")
  .action(shellInit);

program.parse();
