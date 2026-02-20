#!/usr/bin/env node
// ABOUTME: Platform-aware shim that locates and executes the correct tl binary.
// ABOUTME: Supports darwin (arm64/amd64), linux (arm64/amd64), and windows (amd64).

import { spawnSync } from "child_process";
import { existsSync, chmodSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";

const dir =
  typeof __dirname !== "undefined"
    ? __dirname
    : dirname(fileURLToPath(import.meta.url));

const platform = process.platform;
const arch = process.arch;

const archMap = { x64: "amd64", arm64: "arm64" };
const goArch = archMap[arch];

if (!goArch) {
  process.stderr.write(`tl: unsupported architecture: ${arch}\n`);
  process.exit(1);
}

let binaryName;
if (platform === "win32") {
  binaryName = `tl-windows-${goArch}.exe`;
} else if (platform === "darwin") {
  binaryName = `tl-darwin-${goArch}`;
} else if (platform === "linux") {
  binaryName = `tl-linux-${goArch}`;
} else {
  process.stderr.write(`tl: unsupported platform: ${platform}\n`);
  process.exit(1);
}

const binaryPath = join(dir, "..", "binaries", binaryName);

if (!existsSync(binaryPath)) {
  process.stderr.write(
    `tl: no pre-built binary found at ${binaryPath}\n` +
      `tl: supported platforms: darwin-arm64, darwin-amd64, linux-arm64, linux-amd64, windows-amd64\n`,
  );
  process.exit(1);
}

if (platform !== "win32") {
  chmodSync(binaryPath, 0o755);
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
});

if (result.error) {
  process.stderr.write(`tl: failed to run binary: ${result.error.message}\n`);
  process.exit(1);
}

process.exit(result.status ?? 0);
