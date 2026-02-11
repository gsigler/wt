const fs = require("fs");
const path = require("path");
const readline = require("readline");
const { git } = require("../git");
const { writeConfig } = require("../config");

function prompt(rl, question, defaultValue) {
  const suffix = defaultValue ? ` (${defaultValue})` : "";
  return new Promise((resolve) => {
    rl.question(`${question}${suffix}: `, (answer) => {
      resolve(answer.trim() || defaultValue || "");
    });
  });
}

async function init(directory) {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  try {
    const url = await prompt(rl, "Remote URL?");
    if (!url) {
      console.error("A remote URL is required.");
      process.exit(1);
    }

    // Derive directory name from URL if not provided
    if (!directory) {
      directory = path.basename(url, ".git");
    }

    const targetDir = path.resolve(directory);

    if (fs.existsSync(targetDir) && fs.readdirSync(targetDir).length > 0) {
      console.error(`Directory "${directory}" already exists and is not empty.`);
      process.exit(1);
    }

    fs.mkdirSync(targetDir, { recursive: true });

    // Clone bare repo into .bare
    console.log(`\nCloning into ${directory}/.bare ...`);
    git(`clone --bare ${url} .bare`, { cwd: targetDir, stdio: ["pipe", "pipe", "inherit"] });

    // Create .git file pointing to .bare
    fs.writeFileSync(path.join(targetDir, ".git"), "gitdir: .bare\n");

    // Fix the bare repo so worktrees resolve correctly
    // Without this, worktree HEADs point to the wrong relative path
    const bareConfigPath = path.join(targetDir, ".bare", "config");
    let bareConfig = fs.readFileSync(bareConfigPath, "utf-8");
    if (!bareConfig.includes("worktreeConfig")) {
      bareConfig += "\n[extensions]\n\tworktreeConfig = true\n";
    }

    fs.writeFileSync(bareConfigPath, bareConfig);

    // Fix fetch refspec so remote tracking refs work (bare clone defaults
    // to +refs/heads/*:refs/heads/* which breaks origin/branch references)
    git("config remote.origin.fetch +refs/heads/*:refs/remotes/origin/*", {
      gitDir: path.join(targetDir, ".bare"),
    });
    git("fetch origin", {
      gitDir: path.join(targetDir, ".bare"),
      stdio: ["pipe", "pipe", "inherit"],
    });

    // Detect default branch
    let defaultBranch = "main";
    try {
      const headRef = git("symbolic-ref HEAD", {
        gitDir: path.join(targetDir, ".bare"),
      });
      defaultBranch = headRef.replace("refs/heads/", "");
    } catch {
      // fallback to main
    }

    const postCreateScript = await prompt(
      rl,
      "Command to run after creating a worktree?",
      "npm install"
    );

    const copyFilesStr = await prompt(
      rl,
      "Files to copy into each new worktree? (comma-separated)",
      ".env"
    );
    const copyFiles = copyFilesStr
      .split(",")
      .map((f) => f.trim())
      .filter(Boolean);

    const copyDirsStr = await prompt(
      rl,
      "Directories to copy (preserving symlinks)? (comma-separated)",
      "node_modules"
    );
    const copyDirs = copyDirsStr
      .split(",")
      .map((f) => f.trim())
      .filter(Boolean);

    const config = {
      remote: "origin",
      defaultBase: defaultBranch,
      copyFiles,
      copyDirs,
      postCreateScript,
    };

    writeConfig(targetDir, config);

    console.log(`\nProject initialized in ${directory}/`);
    console.log(`  Default branch: ${defaultBranch}`);
    console.log(`  Post-create script: ${postCreateScript || "(none)"}`);
    console.log(`  Copy files: ${copyFiles.join(", ") || "(none)"}`);
    console.log(`  Copy dirs: ${copyDirs.join(", ") || "(none)"}`);
    console.log(`\nNext steps:`);
    console.log(`  cd ${directory}`);
    console.log(`  wt create <branch-name>`);
  } finally {
    rl.close();
  }
}

module.exports = init;
