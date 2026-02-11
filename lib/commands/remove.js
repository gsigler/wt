const path = require("path");
const readline = require("readline");
const { gitInBare } = require("../git");
const { loadConfig } = require("../config");

async function remove(branch, opts) {
  const { root } = loadConfig();
  const worktreePath = path.join(root, branch);
  const forceFlag = opts.force ? " --force" : "";

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
    rl.question(`Delete branch "${branch}" as well? (y/N): `, resolve);
  });
  rl.close();

  if (answer.trim().toLowerCase() === "y") {
    try {
      gitInBare(`branch -d ${branch}`, root);
      console.log(`Branch "${branch}" deleted.`);
    } catch {
      console.error(
        `Could not delete branch "${branch}". It may not be fully merged.`
      );
      console.error(`Use \`git branch -D ${branch}\` to force delete.`);
    }
  }

  console.log("Done.");
}

module.exports = remove;
