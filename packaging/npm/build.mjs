// Assembles the npm packages from the Go binaries in dist/.
//
//   node packaging/npm/build.mjs 1.2.3  ->  packaging/npm/out/cuck-code (all platforms in one package)
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const version = (process.argv[2] || "").replace(/^v/, "");
if (!/^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$/.test(version)) {
  console.error("usage: node packaging/npm/build.mjs <semver>");
  process.exit(2);
}

const targets = [
  { goos: "linux", goarch: "amd64", os: "linux", cpu: "x64" },
  { goos: "linux", goarch: "arm64", os: "linux", cpu: "arm64" },
  { goos: "darwin", goarch: "amd64", os: "darwin", cpu: "x64" },
  { goos: "darwin", goarch: "arm64", os: "darwin", cpu: "arm64" },
  { goos: "windows", goarch: "amd64", os: "win32", cpu: "x64" },
  { goos: "windows", goarch: "arm64", os: "win32", cpu: "arm64" },
];

const main = JSON.parse(fs.readFileSync(path.join(root, "packaging/npm/cuck-code/package.json"), "utf8"));
const out = path.join(root, "packaging/npm/out");
fs.rmSync(out, { recursive: true, force: true });

// One package, every platform's binary inside it under bin/<os>-<cpu>/.
const mainDir = path.join(out, "cuck-code");
fs.cpSync(path.join(root, "packaging/npm/cuck-code"), mainDir, { recursive: true });
for (const t of targets) {
  const exe = t.os === "win32" ? "cuck.exe" : "cuck";
  const src = path.join(root, "dist", `cuck-${t.goos}-${t.goarch}${t.os === "win32" ? ".exe" : ""}`);
  if (!fs.existsSync(src)) {
    console.error(`missing ${path.relative(root, src)}; run \`make dist\` first`);
    process.exit(1);
  }
  const dst = path.join(mainDir, "bin", `${t.os}-${t.cpu}`, exe);
  fs.mkdirSync(path.dirname(dst), { recursive: true });
  fs.copyFileSync(src, dst);
  fs.chmodSync(dst, 0o755);
}
main.version = version;
delete main.optionalDependencies;
fs.writeFileSync(path.join(mainDir, "package.json"), JSON.stringify(main, null, 2) + "\n");
fs.copyFileSync(path.join(root, "README.md"), path.join(mainDir, "README.md"));
fs.copyFileSync(path.join(root, "LICENSE"), path.join(mainDir, "LICENSE"));
console.log(`npm package cuck-code@${version} written to ${path.relative(root, mainDir)}/`);
