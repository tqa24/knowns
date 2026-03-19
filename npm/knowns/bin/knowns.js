#!/usr/bin/env node

const { execFileSync } = require("child_process");
const path = require("path");
const os = require("os");
const fs = require("fs");
const Module = require("module");

function uniq(values) {
  return [...new Set(values)];
}

function resolveFromPackageDir(pkgDir, ext) {
  for (const name of [`knowns${ext}`, "knowns"]) {
    const candidate = path.join(pkgDir, name);
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }
  return null;
}

function getBinaryPath() {
  const platform = os.platform();
  const arch = os.arch();

  const platformMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "win",
  };

  const archMap = {
    arm64: "arm64",
    x64: "x64",
    ia32: "x64", // fallback
  };

  const p = platformMap[platform];
  const a = archMap[arch];

  if (!p || !a) {
    console.error(`Unsupported platform: ${platform}-${arch}`);
    process.exit(1);
  }

  const pkgName = `@knowns/${p}-${a}`;
  const ext = platform === "win32" ? ".exe" : "";
  const pkgParts = pkgName.split("/");

  const packageDirs = uniq([
    path.resolve(__dirname, "..", "node_modules", ...pkgParts),
    path.resolve(__dirname, "..", "..", ...pkgParts),
    path.resolve(__dirname, "..", "..", "node_modules", ...pkgParts),
    path.resolve(__dirname, "..", "..", "..", "node_modules", ...pkgParts),
    ...module.paths.map((base) => path.join(base, ...pkgParts)),
    ...Module.globalPaths.map((base) => path.join(base, ...pkgParts)),
  ]);

  for (const pkgDir of packageDirs) {
    const resolved = resolveFromPackageDir(pkgDir, ext);
    if (resolved) {
      return resolved;
    }
  }

  for (const base of uniq([__dirname, process.cwd(), ...module.paths, ...Module.globalPaths])) {
    try {
      const pkgJson = require.resolve(`${pkgName}/package.json`, { paths: [base] });
      const resolved = resolveFromPackageDir(path.dirname(pkgJson), ext);
      if (resolved) {
        return resolved;
      }
    } catch {}
  }

  try {
    const pkgJson = require.resolve(`${pkgName}/package.json`);
    const resolved = resolveFromPackageDir(path.dirname(pkgJson), ext);
    if (resolved) {
      return resolved;
    }
  } catch {}

  console.error(
    `Could not find knowns binary for ${platform}-${arch}.\n` +
      `Expected package: ${pkgName}\n` +
      `Try reinstalling: npm install knowns`
  );
  process.exit(1);
}

const binary = getBinaryPath();
const args = process.argv.slice(2);

try {
  execFileSync(binary, args, {
    stdio: "inherit",
    env: process.env,
  });
} catch (err) {
  if (err.status !== undefined) {
    process.exit(err.status);
  }
  throw err;
}
