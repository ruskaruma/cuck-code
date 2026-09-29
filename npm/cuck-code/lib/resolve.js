"use strict";

// Each platform's native binary ships in its own package, installed as an
// optional dependency; npm only downloads the one matching this machine.
const PACKAGES = {
  "linux x64": "cuck-code-linux-x64",
  "linux arm64": "cuck-code-linux-arm64",
  "darwin x64": "cuck-code-darwin-x64",
  "darwin arm64": "cuck-code-darwin-arm64",
  "win32 x64": "cuck-code-win32-x64",
  "win32 arm64": "cuck-code-win32-arm64",
};

function binaryPath() {
  const key = `${process.platform} ${process.arch}`;
  const pkg = PACKAGES[key];
  if (!pkg) {
    throw new Error(`cuck-code: ${key} is not supported. Supported: ${Object.keys(PACKAGES).join(", ")}.`);
  }
  const exe = process.platform === "win32" ? "cuck.exe" : "cuck";
  try {
    return require.resolve(`${pkg}/bin/${exe}`);
  } catch {
    throw new Error(
      `cuck-code: the ${pkg} package is missing. It was probably skipped by --omit=optional ` +
        `or --no-optional; reinstall with: npm install -g cuck-code`
    );
  }
}

module.exports = { binaryPath, PACKAGES };
