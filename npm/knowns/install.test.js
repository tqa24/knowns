"use strict";

const { test, expect } = require("bun:test");
const fs = require("node:fs");
const path = require("node:path");

const { getPlatformPackage } = require("./install.js");

function readPackageJson(relativePath) {
  return JSON.parse(fs.readFileSync(path.join(__dirname, relativePath), "utf8"));
}

test("maps Windows x64 to the published package name", () => {
  expect(getPlatformPackage("win32", "x64")).toEqual({
    name: "@knowns/win-x64",
    asset: "knowns-win-x64",
    ext: ".exe",
    packageOs: "win32",
    packageCpu: "x64",
  });
});

test("Windows platform packages use npm's win32 os identifier", () => {
  const x64Pkg = readPackageJson("../knowns-win-x64/package.json");
  const arm64Pkg = readPackageJson("../knowns-win-arm64/package.json");

  expect(x64Pkg.os).toEqual(["win32"]);
  expect(arm64Pkg.os).toEqual(["win32"]);
  expect(x64Pkg.cpu).toEqual(["x64"]);
  expect(arm64Pkg.cpu).toEqual(["arm64"]);
});
