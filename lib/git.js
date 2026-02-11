const { execSync } = require("child_process");

function git(args, opts = {}) {
  const cmd = `git ${args}`;
  return execSync(cmd, {
    encoding: "utf-8",
    stdio: opts.stdio || ["pipe", "pipe", "pipe"],
    cwd: opts.cwd,
    env: { ...process.env, GIT_DIR: opts.gitDir },
  }).trim();
}

function gitInBare(args, projectRoot) {
  const path = require("path");
  return git(args, { gitDir: path.join(projectRoot, ".bare") });
}

module.exports = { git, gitInBare };
