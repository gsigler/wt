const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const path = require("path");
const readline = require("readline");
const gitModule = require("../lib/git");
const configModule = require("../lib/config");

const ROOT = "/projects/myrepo";

function setup(t, { answer = "n", gitInBareImpl } = {}) {
  t.mock.method(configModule, "loadConfig", () => ({
    root: ROOT,
    config: {},
  }));
  t.mock.method(gitModule, "gitInBare", gitInBareImpl || (() => ""));
  t.mock.method(readline, "createInterface", () => ({
    question: (_q, cb) => cb(answer),
    close: () => {},
  }));
  t.mock.method(console, "log", () => {});
  t.mock.method(console, "error", () => {});
  t.mock.method(process, "exit", (code) => {
    throw new Error(`exit(${code})`);
  });

  delete require.cache[require.resolve("../lib/commands/remove")];
  return require("../lib/commands/remove");
}

describe("remove", () => {
  it("removes worktree and skips branch deletion when user declines", async (t) => {
    const remove = setup(t, { answer: "n" });
    await remove("my-branch", { force: false });

    const gitCalls = gitModule.gitInBare.mock.calls.map(
      (c) => c.arguments[0]
    );
    assert.equal(gitCalls.length, 1);
    assert.ok(gitCalls[0].includes("worktree remove"));
    assert.ok(gitCalls[0].includes(path.join(ROOT, "my-branch")));
  });

  it("deletes branch when user confirms", async (t) => {
    const remove = setup(t, { answer: "y" });
    await remove("my-branch", { force: false });

    const gitCalls = gitModule.gitInBare.mock.calls.map(
      (c) => c.arguments[0]
    );
    assert.equal(gitCalls.length, 2);
    assert.equal(gitCalls[1], "branch -d my-branch");
  });

  it("passes --force flag to worktree remove", async (t) => {
    const remove = setup(t, { answer: "n" });
    await remove("my-branch", { force: true });

    const gitCall = gitModule.gitInBare.mock.calls[0].arguments[0];
    assert.ok(gitCall.includes("--force"));
  });

  it("exits on worktree removal failure", async (t) => {
    const remove = setup(t, {
      gitInBareImpl: () => {
        throw new Error("dirty worktree");
      },
    });

    await assert.rejects(() => remove("my-branch", { force: false }), {
      message: "exit(1)",
    });
  });

  it("suggests --force on failure without force flag", async (t) => {
    const remove = setup(t, {
      gitInBareImpl: () => {
        throw new Error("dirty worktree");
      },
    });

    await assert.rejects(() => remove("my-branch", { force: false }));

    const errorMessages = console.error.mock.calls.map(
      (c) => c.arguments[0]
    );
    assert.ok(errorMessages.some((m) => m.includes("--force")));
  });

  it("handles branch deletion failure gracefully", async (t) => {
    let callCount = 0;
    const remove = setup(t, {
      answer: "y",
      gitInBareImpl: () => {
        callCount++;
        if (callCount === 2) throw new Error("not fully merged");
        return "";
      },
    });

    await remove("my-branch", { force: false });

    const errorMessages = console.error.mock.calls.map(
      (c) => c.arguments[0]
    );
    assert.ok(errorMessages.some((m) => m.includes("not be fully merged")));
  });
});
