const { describe, it } = require("node:test");
const assert = require("node:assert/strict");

describe("shell-init", () => {
  it("outputs a wt() function definition", (t) => {
    let output = "";
    t.mock.method(console, "log", (s) => {
      output += s;
    });

    delete require.cache[require.resolve("../lib/commands/shell-init")];
    const shellInit = require("../lib/commands/shell-init");

    shellInit();

    assert.match(output, /wt\(\)/);
  });

  it("auto-cds after create and pr", (t) => {
    let output = "";
    t.mock.method(console, "log", (s) => {
      output += s;
    });

    delete require.cache[require.resolve("../lib/commands/shell-init")];
    const shellInit = require("../lib/commands/shell-init");

    shellInit();

    assert.match(output, /"create"/);
    assert.match(output, /"pr"/);
    assert.match(output, /command wt cd "\$name"/);
  });

  it("uses 'command wt' to avoid recursion", (t) => {
    let output = "";
    t.mock.method(console, "log", (s) => {
      output += s;
    });

    delete require.cache[require.resolve("../lib/commands/shell-init")];
    const shellInit = require("../lib/commands/shell-init");

    shellInit();

    assert.match(output, /command wt/);
  });
});
