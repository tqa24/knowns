export interface CodeSymbol {
  name: string;
  kind: string;
  docPath: string;
  content: string;
  signature: string;
  source: string;
}

export interface CodeEdge {
  from: string;
  to: string;
  type: string;
  fromPath: string;
  toPath: string;
  rawTarget: string;
  targetName: string;
  targetQualifier: string;
  targetModuleHint: string;
  receiverTypeHint: string;
  resolutionStatus: string;
  resolutionConfidence: string;
  resolvedTo: string;
}

export type SupportedLang = "go" | "javascript" | "typescript" | "tsx" | "python";

function chunkID(docPath: string, name: string): string {
  if (!name) return `code::${docPath}::__file__`;
  return `code::${docPath}::${name}`;
}

function fileSymbol(docPath: string, source: string): CodeSymbol {
  return {
    name: "",
    kind: "file",
    docPath,
    content: "",
    signature: "",
    source,
  };
}

import { extractSymbolsAndEdgesImpl } from "./symbols.ts";

export function extractSymbolsAndEdges(
  docPath: string,
  language: SupportedLang,
  source: string,
  root: any,
): { symbols: CodeSymbol[]; edges: CodeEdge[] } {
  if (!root) return { symbols: [fileSymbol(docPath, source)], edges: [] };
  return extractSymbolsAndEdgesImpl(docPath, language, source, root);
}

export { chunkID };
