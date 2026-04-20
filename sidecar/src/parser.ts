import Parser from "web-tree-sitter";

import goWasm from "../node_modules/tree-sitter-wasms/out/tree-sitter-go.wasm" with { type: "file" };
import jsWasm from "../node_modules/tree-sitter-wasms/out/tree-sitter-javascript.wasm" with { type: "file" };
import tsWasm from "../node_modules/tree-sitter-wasms/out/tree-sitter-typescript.wasm" with { type: "file" };
import tsxWasm from "../node_modules/tree-sitter-wasms/out/tree-sitter-tsx.wasm" with { type: "file" };
import pyWasm from "../node_modules/tree-sitter-wasms/out/tree-sitter-python.wasm" with { type: "file" };
import coreWasm from "../node_modules/web-tree-sitter/tree-sitter.wasm" with { type: "file" };

import { extractSymbolsAndEdges, type CodeSymbol, type CodeEdge } from "./extract.ts";

export type SupportedLang = "go" | "javascript" | "typescript" | "tsx" | "python";

const WASM_PATHS: Record<SupportedLang, string> = {
  go: goWasm,
  javascript: jsWasm,
  typescript: tsWasm,
  tsx: tsxWasm,
  python: pyWasm,
};

const langCache = new Map<SupportedLang, any>();
let coreInitialized = false;

async function ensureCore(): Promise<void> {
  if (coreInitialized) return;
  await Parser.init({
    locateFile: () => coreWasm,
  });
  coreInitialized = true;
}

async function loadLanguage(lang: SupportedLang): Promise<any> {
  let cached = langCache.get(lang);
  if (cached) return cached;
  const path = WASM_PATHS[lang];
  if (!path) throw new Error(`unsupported language: ${lang}`);
  const language = await (Parser as any).Language.load(path);
  langCache.set(lang, language);
  return language;
}

export interface ParseResult {
  symbols: CodeSymbol[];
  edges: CodeEdge[];
}

export async function parseSource(
  docPath: string,
  language: SupportedLang,
  source: string,
): Promise<ParseResult> {
  await ensureCore();
  const lang = await loadLanguage(language);
  const parser = new Parser();
  try {
    parser.setLanguage(lang);
    const tree = parser.parse(source);
    if (!tree || !tree.rootNode) {
      return { symbols: [], edges: [] };
    }
    try {
      return extractSymbolsAndEdges(docPath, language, source, tree.rootNode);
    } finally {
      tree.delete();
    }
  } finally {
    parser.delete();
  }
}
