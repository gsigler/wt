const { gitInBare } = require("../git");
const { loadConfig } = require("../config");

function list() {
  const { root } = loadConfig();
  const output = gitInBare("worktree list", root);
  console.log(output);
}

module.exports = list;
