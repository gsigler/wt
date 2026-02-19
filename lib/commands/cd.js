const path = require("path");
const { gitInBare } = require("../git");
const { loadConfig } = require("../config");

function parseWorktreeList(output) {
  const entries = [];
  for (const line of output.split("\n")) {
    if (!line.trim()) continue;
    const match = line.match(/^(\S+)\s+\S+\s+\[(.+)\]$/);
    if (!match) continue;
    const dir = match[1];
    const branch = match[2];
    if (path.basename(dir) === ".bare") continue;
    entries.push({ dir, branch });
  }
  return entries;
}

function cd(name) {
  const { root } = loadConfig();

  if (!name) {
    process.stdout.write(root + "\n");
    return;
  }

  const output = gitInBare("worktree list", root);
  const entries = parseWorktreeList(output);

  // 1. Exact branch name match
  let matches = entries.filter((e) => e.branch === name);
  if (matches.length === 1) {
    process.stdout.write(matches[0].dir + "\n");
    return;
  }

  // 2. Exact directory basename match
  matches = entries.filter((e) => path.basename(e.dir) === name);
  if (matches.length === 1) {
    process.stdout.write(matches[0].dir + "\n");
    return;
  }

  // 3. Exact relative path match (relative to project root)
  matches = entries.filter((e) => path.relative(root, e.dir) === name);
  if (matches.length === 1) {
    process.stdout.write(matches[0].dir + "\n");
    return;
  }

  // 4. Substring match on branch name
  matches = entries.filter((e) => e.branch.includes(name));
  if (matches.length === 1) {
    process.stdout.write(matches[0].dir + "\n");
    return;
  }

  if (matches.length === 0) {
    process.stderr.write(`No worktree found matching "${name}"\n`);
    process.exit(1);
  }

  process.stderr.write(`Multiple worktrees match "${name}":\n`);
  for (const m of matches) {
    process.stderr.write(`  ${m.branch} → ${m.dir}\n`);
  }
  process.exit(1);
}

module.exports = cd;
