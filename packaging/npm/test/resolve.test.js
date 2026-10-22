"use strict";

// Run with: node --test packaging/npm/test/*.test.js
// (Node 22 does not search a bare directory argument; the shell expands the glob.)
const test = require("node:test");
const assert = require("node:assert/strict");
const { replacesLauncher } = require("../cuck-code/lib/resolve.js");

// npm_config_user_agent as each package manager sets it for a postinstall
// (pnpm values captured from `pnpm add -g` + `pnpm approve-builds -g`).
const UA = {
  npm: "npm/10.9.9 node/v22.23.4 linux x64 workspaces/false",
  yarn1: "yarn/1.22.22 npm/? node/v22.23.4 linux x64",
  yarn4: "yarn/4.9.2 npm/? node/v22.23.4 linux x64",
  pnpm10: "pnpm/10.34.7 npm/? node/v22.23.4 linux x64",
  pnpm11: "pnpm/11.28.7 npm/? node/v22.23.4 linux x64",
  pnpm12: "pnpm/12.14.0 npm/? node/? linux x64",
};

test("swaps in the native binary for npm and yarn on macOS and Linux", () => {
  for (const platform of ["linux", "darwin"]) {
    for (const ua of [UA.npm, UA.yarn1, UA.yarn4]) {
      assert.equal(replacesLauncher({ npm_config_user_agent: ua }, platform), true, `${platform} ${ua}`);
    }
  }
});

test("swaps when the user agent is missing or empty", () => {
  assert.equal(replacesLauncher({}, "linux"), true);
  assert.equal(replacesLauncher({ npm_config_user_agent: "" }, "linux"), true);
  assert.equal(replacesLauncher({ npm_config_user_agent: undefined }, "darwin"), true);
});

test("keeps the JS launcher under pnpm, whose shims run bin/cuck with node", () => {
  for (const platform of ["linux", "darwin"]) {
    for (const ua of [UA.pnpm10, UA.pnpm11, UA.pnpm12]) {
      assert.equal(replacesLauncher({ npm_config_user_agent: ua }, platform), false, `${platform} ${ua}`);
    }
  }
});

test("never swaps on Windows", () => {
  for (const ua of [undefined, UA.npm, UA.yarn4, UA.pnpm10]) {
    assert.equal(replacesLauncher({ npm_config_user_agent: ua }, "win32"), false, String(ua));
  }
});
