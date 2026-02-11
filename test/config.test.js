const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const path = require("path");

function freshConfig(t) {
  delete require.cache[require.resolve("../lib/config")];
  return require("../lib/config");
}

describe("findProjectRoot()", () => {
  it("returns dir when worktree.json exists there", (t) => {
    t.mock.method(fs, "existsSync", (p) =>
      p === path.join("/projects/myrepo", "worktree.json")
    );
    const { findProjectRoot } = freshConfig(t);
    assert.equal(findProjectRoot("/projects/myrepo"), "/projects/myrepo");
  });

  it("walks up directories to find worktree.json", (t) => {
    t.mock.method(fs, "existsSync", (p) =>
      p === path.join("/projects/myrepo", "worktree.json")
    );
    const { findProjectRoot } = freshConfig(t);
    assert.equal(
      findProjectRoot("/projects/myrepo/feature-1"),
      "/projects/myrepo"
    );
  });

  it("returns null at filesystem root", (t) => {
    t.mock.method(fs, "existsSync", () => false);
    const { findProjectRoot } = freshConfig(t);
    assert.equal(findProjectRoot("/some/deep/path"), null);
  });
});

describe("loadConfig()", () => {
  it("reads and parses config when found", (t) => {
    const configData = { remote: "origin", defaultBase: "main" };
    t.mock.method(fs, "existsSync", (p) =>
      p === path.join("/projects/myrepo", "worktree.json")
    );
    t.mock.method(process, "cwd", () => "/projects/myrepo");
    // Re-require before mocking readFileSync so Node's module loader can read the file
    const { loadConfig } = freshConfig(t);
    t.mock.method(fs, "readFileSync", () => JSON.stringify(configData));

    const result = loadConfig();
    assert.equal(result.root, "/projects/myrepo");
    assert.deepEqual(result.config, configData);
  });

  it("exits with code 1 when not in a wt project", (t) => {
    t.mock.method(fs, "existsSync", () => false);
    t.mock.method(console, "error", () => {});
    t.mock.method(process, "cwd", () => "/nowhere");
    t.mock.method(process, "exit", (code) => {
      throw new Error(`exit(${code})`);
    });
    const { loadConfig } = freshConfig(t);

    assert.throws(() => loadConfig(), { message: "exit(1)" });
  });
});

describe("writeConfig()", () => {
  it("writes JSON with 2-space indent and trailing newline", (t) => {
    t.mock.method(fs, "writeFileSync", () => {});
    const { writeConfig } = freshConfig(t);

    const config = { remote: "origin", defaultBase: "main" };
    writeConfig("/projects/myrepo", config);

    const [filePath, content] = fs.writeFileSync.mock.calls[0].arguments;
    assert.equal(filePath, path.join("/projects/myrepo", "worktree.json"));
    assert.equal(content, JSON.stringify(config, null, 2) + "\n");
  });
});
