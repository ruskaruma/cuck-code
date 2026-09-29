"use strict";

// postinstall: swap the JS launcher for the native binary, then ask whether
// to hook the user's shells ("Do you really want to get cucked?"). The
// question is asked on the terminal directly, because npm hides script
// output. Nothing here may ever fail the install.
const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");
const { binaryPath } = require("./lib/resolve.js");

function main() {
  let bin;
  try {
    bin = binaryPath();
  } catch (err) {
    console.warn(err.message);
    return;
  }

  if (process.platform !== "win32") {
    // Running the binary directly starts instantly and gives it the signals
    // and exit codes, with no Node process in between.
    const launcher = path.join(__dirname, "bin", "cuck");
    const tmp = `${launcher}.${process.pid}.tmp`;
    try {
      fs.copyFileSync(bin, tmp);
      fs.chmodSync(tmp, 0o755);
      fs.renameSync(tmp, launcher);
      bin = launcher;
    } catch {
      try {
        fs.unlinkSync(tmp);
      } catch {}
    }
  }

  spawnSync(bin, ["setup", "--npm"], { stdio: "inherit" });
}

try {
  main();
} catch {}
