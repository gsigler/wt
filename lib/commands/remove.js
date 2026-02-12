const fs = require("fs");
const path = require("path");
const readline = require("readline");
const { gitInBare } = require("../git");
const { loadConfig } = require("../config");

async function remove(branch, opts) {
  const { root } = loadConfig();
  const worktreePath = path.join(root, branch);
  const forceFlag = opts.force ? " --force" : "";

  // Resolve the actual git branch name from the worktree before removing it
  // (may differ from the directory name, e.g. prs/123 -> feature-xyz)
  let branchName = branch;
  try {
    const dotGit = fs.readFileSync(path.join(worktreePath, ".git"), "utf-8")
      .replace("gitdir: ", "").trim();
    const head = fs.readFileSync(path.join(dotGit, "HEAD"), "utf-8").trim();
    const match = head.match(/^ref: refs\/heads\/(.+)$/);
    if (match) branchName = match[1];
  } catch {}

  // Remove worktree
  console.log(`Removing worktree "${branch}"...`);
  try {
    gitInBare(`worktree remove ${worktreePath}${forceFlag}`, root);
  } catch (err) {
    console.error(`Failed to remove worktree: ${err.message}`);
    if (!opts.force) {
      console.error("Use --force to remove a dirty worktree.");
    }
    process.exit(1);
  }

  // Ask whether to delete the branch
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  const answer = await new Promise((resolve) => {
    rl.question(`Delete branch "${branchName}" as well? (y/N): `, resolve);
  });
  rl.close();

  if (answer.trim().toLowerCase() === "y") {
    try {
      gitInBare(`branch -d ${branchName}`, root);
      console.log(`Branch "${branchName}" deleted.`);
    } catch {
      console.error(
        `Could not delete branch "${branchName}". It may not be fully merged.`
      );
      console.error(`Use \`git branch -D ${branchName}\` to force delete.`);
    }
  }

  console.log("Done.");
}

module.exports = remove;
