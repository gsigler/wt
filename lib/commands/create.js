const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const { gitInBare } = require("../git");
const { loadConfig } = require("../config");

function create(branch, opts) {
  const { root, config } = loadConfig();
  const { remote, defaultBase, copyFiles, copyDirs, postCreateScript } = config;
  const base = opts.base || defaultBase;
  const worktreePath = path.join(root, branch);

  if (fs.existsSync(worktreePath)) {
    console.error(`Directory "${branch}" already exists.`);
    process.exit(1);
  }

  // Fetch latest from remote
  console.log(`Fetching from ${remote}...`);
  gitInBare(`fetch ${remote}`, root);

  // Check if branch already exists
  let branchExists = false;
  try {
    gitInBare(`rev-parse --verify refs/heads/${branch}`, root);
    branchExists = true;
  } catch {}

  // Create worktree, reusing existing branch or creating a new one
  console.log(`Creating worktree for "${branch}" based on ${remote}/${base}...`);
  if (branchExists) {
    gitInBare(`worktree add ${worktreePath} ${branch}`, root);
  } else {
    gitInBare(
      `worktree add ${worktreePath} -b ${branch} --no-track ${remote}/${base}`,
      root
    );
  }

  // Allow git operations in this worktree (shared config has core.bare = true)
  const wtGitDir = fs.readFileSync(path.join(worktreePath, ".git"), "utf-8")
    .replace("gitdir: ", "").trim();
  fs.writeFileSync(
    path.join(wtGitDir, "config.worktree"),
    "[core]\n\tbare = false\n[push]\n\tdefault = current\n\tautoSetupRemote = true\n"
  );

  // Find source worktree to copy from (base branch worktree)
  const sourceWorktree = path.join(root, base);
  const hasSource = fs.existsSync(sourceWorktree);

  // Copy files
  for (const file of copyFiles || []) {
    const src = hasSource ? path.join(sourceWorktree, file) : path.join(root, file);
    const dest = path.join(worktreePath, file);
    if (fs.existsSync(src)) {
      fs.mkdirSync(path.dirname(dest), { recursive: true });
      fs.copyFileSync(src, dest);
      console.log(`Copied ${file}`);
    }
  }

  // Copy directories (preserving symlinks for pnpm, etc.)
  for (const dir of copyDirs || []) {
    const src = hasSource ? path.join(sourceWorktree, dir) : path.join(root, dir);
    const dest = path.join(worktreePath, dir);
    if (fs.existsSync(src)) {
      console.log(`Copying ${dir}...`);
      execSync(`cp -rP ${JSON.stringify(src)} ${JSON.stringify(dest)}`, {
        stdio: "inherit",
      });
      console.log(`Copied ${dir}`);
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
  console.log(`  Based on: ${remote}/${base}`);

  console.log(`\n  cd ${branch}`);
}

module.exports = create;
