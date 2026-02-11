const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const gitModule = require("../lib/git");
const configModule = require("../lib/config");

describe("list", () => {
  it("calls gitInBare with 'worktree list' and logs output", (t) => {
    const worktreeOutput =
      "/projects/myrepo/main  abc1234 [main]\n/projects/myrepo/feat  def5678 [feat]";

    t.mock.method(configModule, "loadConfig", () => ({
      root: "/projects/myrepo",
      config: {},
    }));
    t.mock.method(gitModule, "gitInBare", () => worktreeOutput);
    t.mock.method(console, "log", () => {});

    delete require.cache[require.resolve("../lib/commands/list")];
    const list = require("../lib/commands/list");

    list();

    assert.equal(
      gitModule.gitInBare.mock.calls[0].arguments[0],
      "worktree list"
    );
    assert.equal(
      gitModule.gitInBare.mock.calls[0].arguments[1],
      "/projects/myrepo"
    );
    assert.equal(console.log.mock.calls[0].arguments[0], worktreeOutput);
  });
});
