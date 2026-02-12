const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const { gitInBare } = require("../git");
const { loadConfig } = require("../config");
const { setupWorktree } = require("./create");

function pr(number) {
  // Get PR branch name from GitHub CLI
  let branch;
  try {
    branch = execSync(
      `gh pr view ${number} --json headRefName -q .headRefName`,
      { encoding: "utf-8", stdio: ["pipe", "pipe", "pipe"] }
    ).trim();
  } catch {
    console.error(`Failed to get PR #${number}. Is \`gh\` installed and authenticated?`);
    process.exit(1);
  }

  const { root, config } = loadConfig();
  const { remote, defaultBase } = config;
  const base = defaultBase;
  const prsDir = path.join(root, "prs");
  const worktreePath = path.join(prsDir, String(number));

  if (fs.existsSync(worktreePath)) {
    console.error(`Directory "prs/${number}" already exists.`);
    process.exit(1);
  }

  // Fetch latest from remote
  console.log(`Fetching from ${remote}...`);
  gitInBare(`fetch ${remote}`, root);

  // Delete stale local branch if it exists (may be left over from a previous
  // attempt or point at the wrong commits). For PRs we always want to match
  // the remote.
  try {
    gitInBare(`rev-parse --verify refs/heads/${branch}`, root);
    console.log(`Deleting stale local branch "${branch}"...`);
    gitInBare(`branch -D ${branch}`, root);
  } catch {}

  // Ensure prs/ directory exists
  fs.mkdirSync(prsDir, { recursive: true });

  // Create worktree from the PR's remote branch
  console.log(`Creating worktree for PR #${number} (${branch}) from ${remote}/${branch}...`);
  gitInBare(
    `worktree add ${worktreePath} -b ${branch} --no-track ${remote}/${branch}`,
    root
  );

  setupWorktree(root, config, worktreePath, base);

  console.log(`\nWorktree ready at ./prs/${number}`);
  console.log(`  PR: #${number}`);
  console.log(`  Branch: ${branch}`);

  console.log(`\n  cd prs/${number}`);
}

module.exports = pr;
