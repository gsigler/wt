const fs = require("fs");
const path = require("path");

const CONFIG_FILE = "worktree.json";

function findProjectRoot(startDir = process.cwd()) {
  let dir = startDir;
  while (true) {
    if (fs.existsSync(path.join(dir, CONFIG_FILE))) {
      return dir;
    }
    const parent = path.dirname(dir);
    if (parent === dir) {
      return null;
    }
    dir = parent;
  }
}

function loadConfig() {
  const root = findProjectRoot();
  if (!root) {
    console.error(
      "Not inside a wt project. Run `wt init` first, or cd into a project directory."
    );
    process.exit(1);
  }
  const configPath = path.join(root, CONFIG_FILE);
  const config = JSON.parse(fs.readFileSync(configPath, "utf-8"));
  return { root, config };
}

function writeConfig(dir, config) {
  const configPath = path.join(dir, CONFIG_FILE);
  fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + "\n");
}

module.exports = { findProjectRoot, loadConfig, writeConfig, CONFIG_FILE };
