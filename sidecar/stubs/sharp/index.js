// Stub for sharp — image processing not used by text embedding pipeline.
function sharp() {
  throw new Error("sharp stub: image processing not available in text-only sidecar");
}
sharp.cache = () => sharp;
sharp.simd = () => false;
sharp.format = {};
sharp.versions = {};
module.exports = sharp;
module.exports.default = sharp;
