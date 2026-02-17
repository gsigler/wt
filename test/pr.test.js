const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const path = require("path");
const childProcess = require("child_process");
const gitModule = require("../lib/git");
const configModule = require("../lib/config");

const ROOT = "/projects/myrepo";

function setup(t, configOverrides = {}) {
  const config = {
    remote: "origin",
    defaultBase: "main",
    copyFiles: [],
    postCreateScript: null,
    ...configOverrides,
  };

  t.mock.method(configModule, "loadConfig", () => ({ root: ROOT, config }));
  t.mock.method(gitModule, "gitInBare", (args) => {
    if (args.includes("rev-parse")) throw new Error("not found");
    return "";
  });
  t.mock.method(fs, "existsSync", () => false);
  t.mock.method(fs, "cpSync", () => {});
  t.mock.method(fs, "mkdirSync", () => {});
  t.mock.method(childProcess, "execSync", (cmd) => {
    if (String(cmd).startsWith("gh pr view")) return "feature-from-pr\n";
    return "";
  });
  t.mock.method(console, "log", () => {});
  t.mock.method(console, "error", () => {});
  t.mock.method(process, "exit", (code) => {
    throw new Error(`exit(${code})`);
  });

  // Require modules BEFORE mocking readFileSync/writeFileSync
  // (Node's require uses fs.readFileSync internally to load .js files)
  delete require.cache[require.resolve("../lib/commands/pr")];
  delete require.cache[require.resolve("../lib/commands/create")];
  const pr = require("../lib/commands/pr");

  t.mock.method(fs, "readFileSync", (p) => {
    if (String(p).endsWith(".git")) return "gitdir: /fake/gitdir";
    return "";
  });
  t.mock.method(fs, "writeFileSync", () => {});

  return pr;
}

describe("pr", () => {
  it("fetches PR branch name via gh and creates worktree in prs/", (t) => {
    const pr = setup(t);
    pr("123");

    // First execSync call is gh pr view
    const ghCall = childProcess.execSync.mock.calls[0];
    assert.ok(ghCall.arguments[0].includes("gh pr view 123"));

    const gitCalls = gitModule.gitInBare.mock.calls.map((c) => c.arguments[0]);
    assert.equal(gitCalls[0], "fetch origin");
    assert.ok(gitCalls[2].includes("worktree add"));
    assert.ok(gitCalls[2].includes(path.join(ROOT, "prs", "123")));
    assert.ok(gitCalls[2].includes("-b feature-from-pr"));
    assert.ok(
      gitCalls[2].includes("origin/feature-from-pr"),
      "should base on the PR's remote branch, not defaultBase"
    );
  });

  it("creates prs/ directory", (t) => {
    const pr = setup(t);
    pr("456");

    const mkdirCalls = fs.mkdirSync.mock.calls.map((c) => c.arguments[0]);
    assert.ok(mkdirCalls.includes(path.join(ROOT, "prs")));
  });

  it("exits if prs/<number> directory already exists", (t) => {
    const pr = setup(t);
    fs.existsSync.mock.mockImplementation((p) => {
      if (p === path.join(ROOT, "prs", "123")) return true;
      return false;
    });

    assert.throws(() => pr("123"), { message: "exit(1)" });
    const errMsg = console.error.mock.calls[0].arguments[0];
    assert.ok(errMsg.includes("prs/123"));
  });

  it("exits if gh CLI fails", (t) => {
    const pr = setup(t);
    childProcess.execSync.mock.mockImplementation((cmd) => {
      if (String(cmd).startsWith("gh pr view")) throw new Error("gh failed");
      return "";
    });

    assert.throws(() => pr("999"), { message: "exit(1)" });
    const errMsg = console.error.mock.calls[0].arguments[0];
    assert.ok(errMsg.includes("999"));
  });

  it("deletes stale local branch before creating worktree", (t) => {
    const pr = setup(t);
    gitModule.gitInBare.mock.mockImplementation((args) => {
      if (args.includes("rev-parse")) return "abc123";
      return "";
    });

    pr("123");

    const gitCalls = gitModule.gitInBare.mock.calls.map((c) => c.arguments[0]);
    assert.ok(
      gitCalls.includes("branch -D feature-from-pr"),
      "should delete existing local branch"
    );
    const addCall = gitCalls.find((c) => c.includes("worktree add"));
    assert.ok(addCall);
    assert.ok(addCall.includes("-b feature-from-pr"));
    assert.ok(addCall.includes("origin/feature-from-pr"));
  });

  it("runs post-create script in worktree dir", (t) => {
    const pr = setup(t, { postCreateScript: "npm install" });
    pr("123");

    const scriptCall = childProcess.execSync.mock.calls.find(
      (c) => c.arguments[0] === "npm install"
    );
    assert.ok(scriptCall, "should run post-create script");
    assert.equal(
      scriptCall.arguments[1].cwd,
      path.join(ROOT, "prs", "123")
    );
  });

  it("copies configured files", (t) => {
    const pr = setup(t, { copyFiles: [".env"] });
    fs.existsSync.mock.mockImplementation((p) => {
      if (p === path.join(ROOT, "prs", "123")) return false;
      return true;
    });

    pr("123");

    assert.equal(fs.cpSync.mock.callCount(), 1);
    assert.equal(
      fs.cpSync.mock.calls[0].arguments[1],
      path.join(ROOT, "prs", "123", ".env")
    );
  });
});
