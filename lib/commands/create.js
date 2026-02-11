const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const { gitInBare } = require("../git");
const { loadConfig } = require("../config");

function create(branch, opts) {
  const { root, config } = loadConfig();
  const { remote, defaultBase, copyFiles, postCreateScript } = config;
  const base = opts.base || defaultBase;
  const worktreePath = path.join(root, branch);

  if (fs.existsSync(worktreePath)) {
    console.error(`Directory "${branch}" already exists.`);
    process.exit(1);
  }

  // Fetch latest from remote
  console.log(`Fetching from ${remote}...`);
  gitInBare(`fetch ${remote}`, root);

  // Create worktree with new branch tracking remote base
  console.log(`Creating worktree for "${branch}" based on ${remote}/${base}...`);
  gitInBare(
    `worktree add ${worktreePath} -b ${branch} ${remote}/${base}`,
    root
  );

  // Set upstream tracking
  gitInBare(
    `branch --set-upstream-to=${remote}/${base} ${branch}`,
    root
  );

  // Copy files
  for (const file of copyFiles || []) {
    const src = path.join(root, file);
    const dest = path.join(worktreePath, file);
    if (fs.existsSync(src)) {
      fs.mkdirSync(path.dirname(dest), { recursive: true });
      fs.copyFileSync(src, dest);
      console.log(`Copied ${file}`);
    }
  }

  // Run post-create script
  if (postCreateScript) {
    console.log(`Running: ${postCreateScript}`);
    execSync(postCreateScript, {
      cwd: worktreePath,
      stdio: "inherit",
    });
  }

  console.log(`\nWorktree ready at ./${branch}`);
  console.log(`  Branch: ${branch}`);
  console.log(`  Tracking: ${remote}/${base}`);
}

module.exports = create;
