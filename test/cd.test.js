const { describe, it, beforeEach } = require("node:test");
const assert = require("node:assert/strict");
const gitModule = require("../lib/git");
const configModule = require("../lib/config");

const WORKTREE_OUTPUT = [
  "/projects/myrepo/.bare           (bare)",
  "/projects/myrepo/main            abc1234 [main]",
  "/projects/myrepo/feature-branch  def5678 [feature-branch]",
  "/projects/myrepo/prs/123         ghi9012 [fix/login-bug]",
  "/projects/myrepo/prs/456         jkl3456 [feat/signup]",
].join("\n");

function loadCd(t) {
  t.mock.method(configModule, "loadConfig", () => ({
    root: "/projects/myrepo",
    config: {},
  }));
  t.mock.method(gitModule, "gitInBare", () => WORKTREE_OUTPUT);
  delete require.cache[require.resolve("../lib/commands/cd")];
  return require("../lib/commands/cd");
}

describe("cd", () => {
  let stdout;
  let stderr;

  beforeEach((t) => {
    stdout = "";
    stderr = "";
    t.mock.method(process.stdout, "write", (s) => {
      stdout += s;
    });
    t.mock.method(process.stderr, "write", (s) => {
      stderr += s;
    });
  });

  it("prints project root when no name given", (t) => {
    t.mock.method(configModule, "loadConfig", () => ({
      root: "/projects/myrepo",
      config: {},
    }));
    delete require.cache[require.resolve("../lib/commands/cd")];
    const cd = require("../lib/commands/cd");

    cd();

    assert.equal(stdout, "/projects/myrepo\n");
  });

  it("matches exact branch name", (t) => {
    const cd = loadCd(t);
    cd("feature-branch");
    assert.equal(stdout, "/projects/myrepo/feature-branch\n");
  });

  it("matches directory basename for PR worktrees", (t) => {
    const cd = loadCd(t);
    cd("123");
    assert.equal(stdout, "/projects/myrepo/prs/123\n");
  });

  it("matches relative path", (t) => {
    const cd = loadCd(t);
    cd("prs/123");
    assert.equal(stdout, "/projects/myrepo/prs/123\n");
  });

  it("matches substring on branch name", (t) => {
    const cd = loadCd(t);
    cd("login");
    assert.equal(stdout, "/projects/myrepo/prs/123\n");
  });

  it("exits with error when no match found", (t) => {
    const cd = loadCd(t);
    t.mock.method(process, "exit", () => {
      throw new Error("process.exit");
    });

    assert.throws(() => cd("nonexistent"), /process\.exit/);
    assert.match(stderr, /No worktree found matching "nonexistent"/);
  });

  it("exits with error when multiple matches found", (t) => {
    const cd = loadCd(t);
    t.mock.method(process, "exit", () => {
      throw new Error("process.exit");
    });

    assert.throws(() => cd("feat"), /process\.exit/);
    assert.match(stderr, /Multiple worktrees match "feat"/);
  });

  it("excludes .bare entry", (t) => {
    const cd = loadCd(t);
    t.mock.method(process, "exit", () => {
      throw new Error("process.exit");
    });

    assert.throws(() => cd(".bare"), /process\.exit/);
    assert.match(stderr, /No worktree found/);
  });
});
