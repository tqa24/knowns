import { pipeline, env, type FeatureExtractionPipeline } from "@xenova/transformers";

env.allowRemoteModels = true;
env.allowLocalModels = true;

export interface ModelConfig {
  name: string;
  huggingFaceId: string;
  dimensions: number;
  maxTokens: number;
  queryPrefix?: string;
  docPrefix?: string;
}

export class Embedder {
  private pipe: FeatureExtractionPipeline | null = null;
  private config: ModelConfig | null = null;

  async init(config: ModelConfig, cacheDir?: string): Promise<void> {
    if (cacheDir) {
      env.cacheDir = cacheDir;
    }
    this.config = config;
    this.pipe = (await pipeline("feature-extraction", config.huggingFaceId, {
      quantized: true,
    })) as FeatureExtractionPipeline;
  }

  async embed(texts: string[], kind: "query" | "doc"): Promise<number[][]> {
    if (!this.pipe || !this.config) {
      throw new Error("embedder not initialized");
    }
    const prefix = kind === "query" ? this.config.queryPrefix : this.config.docPrefix;
    const inputs = prefix ? texts.map((t) => prefix + t) : texts;
    const output = await this.pipe(inputs, { pooling: "mean", normalize: true });
    const dims = this.config.dimensions;
    const data = output.data as Float32Array;
    const rows: number[][] = [];
    for (let i = 0; i < inputs.length; i++) {
      rows.push(Array.from(data.slice(i * dims, (i + 1) * dims)));
    }
    return rows;
  }

  dimensions(): number {
    return this.config?.dimensions ?? 0;
  }
}
