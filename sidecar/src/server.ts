import { Embedder, type ModelConfig } from "./embedder.ts";
import { parseSource, type SupportedLang } from "./parser.ts";

interface JsonRpcRequest {
  jsonrpc: "2.0";
  id: number | string | null;
  method: string;
  params?: any;
}

interface JsonRpcResponse {
  jsonrpc: "2.0";
  id: number | string | null;
  result?: any;
  error?: { code: number; message: string; data?: any };
}

const embedder = new Embedder();

async function handle(req: JsonRpcRequest): Promise<JsonRpcResponse | null> {
  try {
    switch (req.method) {
      case "ping":
        return ok(req.id, { ok: true, version: "0.1.0" });

      case "init": {
        const cfg = req.params?.model as ModelConfig;
        const cacheDir = req.params?.cacheDir as string | undefined;
        if (!cfg) throw new Error("missing params.model");
        await embedder.init(cfg, cacheDir);
        return ok(req.id, { dimensions: embedder.dimensions() });
      }

      case "embed": {
        const texts = req.params?.texts as string[];
        const kind = (req.params?.kind ?? "doc") as "query" | "doc";
        if (!Array.isArray(texts)) throw new Error("missing params.texts");
        const vectors = await embedder.embed(texts, kind);
        return ok(req.id, { vectors });
      }

      case "parse": {
        const docPath = req.params?.path as string;
        const language = req.params?.language as SupportedLang;
        const source = req.params?.source as string;
        if (typeof docPath !== "string") throw new Error("missing params.path");
        if (typeof language !== "string") throw new Error("missing params.language");
        if (typeof source !== "string") throw new Error("missing params.source");
        const result = await parseSource(docPath, language, source);
        return ok(req.id, result);
      }

      case "shutdown":
        queueMicrotask(() => process.exit(0));
        return ok(req.id, { ok: true });

      default:
        return err(req.id, -32601, `unknown method: ${req.method}`);
    }
  } catch (e) {
    return err(req.id, -32000, (e as Error).message);
  }
}

function ok(id: JsonRpcRequest["id"], result: any): JsonRpcResponse {
  return { jsonrpc: "2.0", id, result };
}

function err(id: JsonRpcRequest["id"], code: number, message: string): JsonRpcResponse {
  return { jsonrpc: "2.0", id, error: { code, message } };
}

function send(resp: JsonRpcResponse): void {
  process.stdout.write(JSON.stringify(resp) + "\n");
}

async function main(): Promise<void> {
  const decoder = new TextDecoder();
  let buffer = "";

  for await (const chunk of process.stdin as any) {
    buffer += decoder.decode(chunk as Uint8Array, { stream: true });
    let nl: number;
    while ((nl = buffer.indexOf("\n")) >= 0) {
      const line = buffer.slice(0, nl).trim();
      buffer = buffer.slice(nl + 1);
      if (!line) continue;
      let req: JsonRpcRequest;
      try {
        req = JSON.parse(line);
      } catch {
        send(err(null, -32700, "parse error"));
        continue;
      }
      const resp = await handle(req);
      if (resp) send(resp);
    }
  }
}

main().catch((e) => {
  process.stderr.write(`fatal: ${(e as Error).message}\n`);
  process.exit(1);
});
