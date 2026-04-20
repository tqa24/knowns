#!/usr/bin/env bun
import { $ } from "bun";
import { mkdir, cp, rm, readdir } from "node:fs/promises";
import { existsSync } from "node:fs";
import { platform, arch } from "node:os";

type OS = "darwin" | "linux" | "win32";
type Arch = "x64" | "arm64";

interface Target {
  bun: string;
  out: string;
  ext: string;
  os: OS;
  arch: Arch;
}

const TARGETS: Target[] = [
  { bun: "bun-darwin-x64", out: "knowns-embed-darwin-x64", ext: "", os: "darwin", arch: "x64" },
  { bun: "bun-darwin-arm64", out: "knowns-embed-darwin-arm64", ext: "", os: "darwin", arch: "arm64" },
  { bun: "bun-linux-x64", out: "knowns-embed-linux-x64", ext: "", os: "linux", arch: "x64" },
  { bun: "bun-linux-arm64", out: "knowns-embed-linux-arm64", ext: "", os: "linux", arch: "arm64" },
  { bun: "bun-windows-x64", out: "knowns-embed-win-x64", ext: ".exe", os: "win32", arch: "x64" },
  { bun: "bun-windows-arm64", out: "knowns-embed-win-arm64", ext: ".exe", os: "win32", arch: "arm64" },
];

const only = process.argv[2];
const strict = process.env.SIDECAR_STRICT === "1";
const hostOS = platform() as OS;
const hostArch = arch() as Arch;

await mkdir("dist", { recursive: true });

const failed: { target: string; error: string }[] = [];
const skipped: { target: string; reason: string }[] = [];

async function copyNatives(target: Target, bundleDir: string) {
  const nativeDir = `node_modules/onnxruntime-node/bin/napi-v3/${target.os}/${target.arch}`;
  if (!existsSync(nativeDir)) {
    throw new Error(`native libs missing: ${nativeDir}`);
  }
  const entries = await readdir(nativeDir);
  for (const entry of entries) {
    await cp(`${nativeDir}/${entry}`, `${bundleDir}/${entry}`);
  }
}

async function patchOnnxNodeAddon(target: Target) {
  // Bun --compile bundles the ORIGINAL onnxruntime_binding.node from node_modules
  // and extracts it to a temp dir at runtime. We must rewrite its @rpath dylib
  // reference to @loader_path BEFORE bun build so the embedded copy is patched.
  // dyld will then resolve the dylib next to the .node file in the bunfs temp dir.
  // But the dylib lives next to the OUTER binary, not in temp. So we use
  // @executable_path which points at the outer Bun binary's directory.
  if (target.os !== "darwin" || hostOS !== "darwin") return;
  const nodePath = `node_modules/onnxruntime-node/bin/napi-v3/${target.os}/${target.arch}/onnxruntime_binding.node`;
  if (!existsSync(nodePath)) return;
  const check = await $`otool -L ${nodePath}`.quiet().nothrow();
  const out = check.stdout.toString();
  if (!out.includes("@rpath/libonnxruntime")) return;
  console.log(`[patch] ${nodePath}: @rpath -> @executable_path`);
  await $`install_name_tool -change @rpath/libonnxruntime.1.14.0.dylib @executable_path/libonnxruntime.1.14.0.dylib ${nodePath}`;
  await $`codesign --force --sign - ${nodePath}`.quiet().nothrow();
}

async function patchBinary(target: Target, binPath: string) {
  // Bun-compiled binaries have a __LINKEDIT layout that install_name_tool/patchelf
  // cannot rewrite. Native lib discovery is handled at spawn time by the Go parent
  // setting DYLD_LIBRARY_PATH / LD_LIBRARY_PATH / PATH to the bundle directory.
  // On macOS we still re-codesign so the binary is launchable on arm64.
  if (target.os === "darwin" && hostOS === "darwin") {
    await $`codesign --remove-signature ${binPath}`.quiet().nothrow();
    await $`codesign --force --sign - ${binPath}`;
  }
  void target;
  void binPath;
}

for (const t of TARGETS) {
  if (only && only !== t.bun && only !== t.out) continue;

  const bundleDir = `dist/${t.out}`;
  const binPath = `${bundleDir}/knowns-embed${t.ext}`;
  console.log(`[build] ${t.bun} -> ${bundleDir}/`);

  try {
    await rm(bundleDir, { recursive: true, force: true });
    await mkdir(bundleDir, { recursive: true });
    await patchOnnxNodeAddon(t);
    await $`bun build ./src/server.ts --compile --target=${t.bun} --outfile=${binPath} --minify`;
    await copyNatives(t, bundleDir);
    await patchBinary(t, binPath);
    console.log(`[ok]    ${t.bun}`);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    if (strict) throw err;
    console.warn(`[skip]  ${t.bun}: ${message.split("\n")[0]}`);
    failed.push({ target: t.bun, error: message });
  }
}

if (failed.length > 0) {
  console.warn(`done with ${failed.length} skipped target(s):`);
  for (const f of failed) console.warn(`  - ${f.target}`);
  console.warn("(set SIDECAR_STRICT=1 to fail the build instead)");
} else {
  console.log("done");
}
