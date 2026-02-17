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
  t.mock.method(childProcess, "execSync", () => "");
  t.mock.method(console, "log", () => {});
  t.mock.method(console, "error", () => {});
  t.mock.method(process, "exit", (code) => {
    throw new Error(`exit(${code})`);
  });

  // Require module BEFORE mocking readFileSync/writeFileSync
  // (Node's require uses fs.readFileSync internally to load .js files)
  delete require.cache[require.resolve("../lib/commands/create")];
  const create = require("../lib/commands/create");

  t.mock.method(fs, "readFileSync", (p) => {
    if (String(p).endsWith(".git")) return "gitdir: /fake/gitdir";
    return "";
  });
  t.mock.method(fs, "writeFileSync", () => {});

  return create;
}

describe("create", () => {
  it("fetches, creates worktree, and sets upstream", (t) => {
    const create = setup(t);
    create("my-branch", {});

    const calls = gitModule.gitInBare.mock.calls.map((c) => c.arguments[0]);
    assert.equal(calls[0], "fetch origin");
    assert.ok(calls[2].includes("worktree add"));
    assert.ok(calls[2].includes(path.join(ROOT, "my-branch")));
    assert.ok(calls[2].includes("-b my-branch"));
    assert.ok(calls[2].includes("origin/main"));
  });

  it("uses custom base branch from opts", (t) => {
    const create = setup(t);
    create("my-branch", { base: "develop" });

    const calls = gitModule.gitInBare.mock.calls.map((c) => c.arguments[0]);
    assert.ok(calls[2].includes("origin/develop"));
  });

  it("exits if branch directory already exists", (t) => {
    const create = setup(t);
    fs.existsSync.mock.mockImplementation(() => true);

    assert.throws(() => create("my-branch", {}), { message: "exit(1)" });
  });

  it("copies files that exist at source", (t) => {
    const create = setup(t, { copyFiles: [".env", "config.json"] });
    fs.existsSync.mock.mockImplementation((p) => {
      if (p === path.join(ROOT, "my-branch")) return false;
      return true;
    });

    create("my-branch", {});

    assert.equal(fs.cpSync.mock.callCount(), 2);
    // Source is base worktree (main) since it exists
    assert.equal(
      fs.cpSync.mock.calls[0].arguments[0],
      path.join(ROOT, "main", ".env")
    );
    assert.equal(
      fs.cpSync.mock.calls[0].arguments[1],
      path.join(ROOT, "my-branch", ".env")
    );
  });

  it("skips missing copy files silently", (t) => {
    const create = setup(t, { copyFiles: [".env"] });
    // existsSync returns false for everything (default)
    create("my-branch", {});
    assert.equal(fs.cpSync.mock.callCount(), 0);
  });

  it("runs post-create script in worktree dir", (t) => {
    const create = setup(t, { postCreateScript: "npm install" });
    create("my-branch", {});

    assert.equal(childProcess.execSync.mock.callCount(), 1);
    assert.equal(
      childProcess.execSync.mock.calls[0].arguments[0],
      "npm install"
    );
    assert.equal(
      childProcess.execSync.mock.calls[0].arguments[1].cwd,
      path.join(ROOT, "my-branch")
    );
  });

  it("skips post-create script when not configured", (t) => {
    const create = setup(t);
    create("my-branch", {});
    assert.equal(childProcess.execSync.mock.callCount(), 0);
  });

  it("writes worktree config to disable bare mode", (t) => {
    const create = setup(t);
    create("my-branch", {});

    const writeCalls = fs.writeFileSync.mock.calls;
    const configCall = writeCalls.find((c) =>
      String(c.arguments[0]).includes("config.worktree")
    );
    assert.ok(configCall, "should write config.worktree");
    assert.ok(configCall.arguments[1].includes("bare = false"));
    assert.ok(configCall.arguments[1].includes("autoSetupRemote = true"));
  });

  it("exports setupWorktree", (t) => {
    const create = setup(t);
    assert.equal(typeof create.setupWorktree, "function");
  });
});
