"use strict";

// All six native binaries ship inside this one package, under
// bin/<platform>-<arch>/; pick the one for this machine.
const fs = require("fs");
const path = require("path");

const PLATFORMS = ["linux-x64", "linux-arm64", "darwin-x64", "darwin-arm64", "win32-x64", "win32-arm64"];

function binaryPath() {
  const key = `${process.platform}-${process.arch}`;
  if (!PLATFORMS.includes(key)) {
    throw new Error(`cuck-code: ${key} is not supported. Supported: ${PLATFORMS.join(", ")}.`);
  }
  const exe = process.platform === "win32" ? "cuck.exe" : "cuck";
  const bin = path.join(__dirname, "..", "bin", key, exe);
  if (!fs.existsSync(bin)) {
    throw new Error(`cuck-code: the ${key} binary is missing from this install; reinstall with: npm install -g cuck-code`);
  }
  return bin;
}

// Whether postinstall should copy the native binary over bin/cuck. Never on
// Windows (the binary is cuck.exe there). Not under pnpm either: its global
// shims run `node <pkg>/bin/cuck`, so once the binary sits there every
// `cuck` call dies in Node with "SyntaxError: Invalid or unexpected token".
// pnpm sets npm_config_user_agent to "pnpm/<version> ..." for build scripts,
// including the ones `pnpm approve-builds -g` runs.
function replacesLauncher(env, platform) {
  const agent = String((env && env.npm_config_user_agent) || "");
  return platform !== "win32" && !agent.startsWith("pnpm/");
}

module.exports = { binaryPath, replacesLauncher, PLATFORMS };
