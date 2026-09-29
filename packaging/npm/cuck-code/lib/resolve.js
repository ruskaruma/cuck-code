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

module.exports = { binaryPath, PLATFORMS };
