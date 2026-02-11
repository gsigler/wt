const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const path = require("path");
const childProcess = require("child_process");

function setup(t, mockReturn = "") {
  t.mock.method(childProcess, "execSync", () => mockReturn);
  delete require.cache[require.resolve("../lib/git")];
  return require("../lib/git");
}

describe("git()", () => {
  it("builds correct command string", (t) => {
    const { git } = setup(t);
    git("status");
    assert.equal(childProcess.execSync.mock.calls[0].arguments[0], "git status");
  });

  it("returns trimmed stdout", (t) => {
    const { git } = setup(t, "  hello world  ");
    assert.equal(git("status"), "hello world");
  });

  it("passes cwd option", (t) => {
    const { git } = setup(t);
    git("log", { cwd: "/some/dir" });
    assert.equal(childProcess.execSync.mock.calls[0].arguments[1].cwd, "/some/dir");
  });

  it("sets GIT_DIR from gitDir option", (t) => {
    const { git } = setup(t);
    git("log", { gitDir: "/repo/.bare" });
    assert.equal(
      childProcess.execSync.mock.calls[0].arguments[1].env.GIT_DIR,
      "/repo/.bare"
    );
  });

  it("defaults to pipe stdio", (t) => {
    const { git } = setup(t);
    git("status");
    assert.deepEqual(childProcess.execSync.mock.calls[0].arguments[1].stdio, [
      "pipe",
      "pipe",
      "pipe",
    ]);
  });

  it("allows overriding stdio", (t) => {
    const { git } = setup(t);
    git("status", { stdio: "inherit" });
    assert.equal(
      childProcess.execSync.mock.calls[0].arguments[1].stdio,
      "inherit"
    );
  });
});

describe("gitInBare()", () => {
  it("sets GIT_DIR to <root>/.bare", (t) => {
    const { gitInBare } = setup(t);
    gitInBare("fetch origin", "/projects/myrepo");
    assert.equal(
      childProcess.execSync.mock.calls[0].arguments[1].env.GIT_DIR,
      path.join("/projects/myrepo", ".bare")
    );
  });

  it("passes args to git command", (t) => {
    const { gitInBare } = setup(t);
    gitInBare("worktree list", "/root");
    assert.equal(
      childProcess.execSync.mock.calls[0].arguments[0],
      "git worktree list"
    );
  });
});
