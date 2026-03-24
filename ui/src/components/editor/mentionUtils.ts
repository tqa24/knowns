import { normalizeKnownsTaskReferences } from "../../lib/knownsReferences";

// Regex patterns for mentions
// Task: supports both numeric IDs (task-42, task-42.1) and alphanumeric IDs (task-pdyd2e, task-4sv3rh)
const TASK_MENTION_REGEX = /@(task-[a-zA-Z0-9]+(?:\.[a-zA-Z0-9]+)?)/g;
// Doc: excludes trailing punctuation (comma, semicolon, etc.) but allows colons for line numbers
const DOC_MENTION_REGEX = /@docs?\/([^\s,;!?"'()]+)/g;
const KNOWNS_DOC_PATH_REGEX = /(^|[\s(])(\.knowns\/docs\/[^\s,;!?"'()]+\.md)\b/g;

/**
 * Normalize doc path - ensure .md extension
 */
export function normalizeDocPath(path: string): string {
  return path.endsWith(".md") ? path : `${path}.md`;
}

export function toDocPath(path: string): string {
  let normalized = path.trim();

  // Strip query params and hash before normalizing
  const queryIndex = normalized.indexOf("?");
  const hashIndex = normalized.indexOf("#");
  const splitIndex = queryIndex >= 0 ? queryIndex : hashIndex >= 0 ? hashIndex : -1;
  let suffix = "";
  if (splitIndex >= 0) {
    suffix = normalized.slice(splitIndex);
    normalized = normalized.slice(0, splitIndex);
  }

  if (normalized.startsWith("@doc/")) {
    normalized = normalized.slice(5);
  } else if (normalized.startsWith("@docs/")) {
    normalized = normalized.slice(6);
  } else if (normalized.startsWith(".knowns/docs/")) {
    normalized = normalized.slice(".knowns/docs/".length);
  } else if (normalized.startsWith("/.knowns/docs/")) {
    normalized = normalized.slice("/.knowns/docs/".length);
  } else if (normalized.startsWith("docs/")) {
    normalized = normalized.slice("docs/".length);
  } else if (normalized.startsWith("/docs/")) {
    normalized = normalized.slice("/docs/".length);
  } else if (normalized.startsWith("/")) {
    normalized = normalized.slice(1);
  }

  // Convert URL-style ?L= suffix back to mention-style :line suffix
  if (suffix) {
    const rangeMatch = suffix.match(/^\?L=(\d+)-(\d+)/);
    const lineMatch = !rangeMatch && suffix.match(/^\?L=(\d+)/);
    const headingMatch = !rangeMatch && !lineMatch && suffix.match(/^#(.+)/);
    if (rangeMatch && rangeMatch[1] && rangeMatch[2]) {
      normalized = `${normalizeDocPath(normalized)}:${rangeMatch[1]}-${rangeMatch[2]}`;
      return normalized;
    }
    if (lineMatch && lineMatch[1]) {
      normalized = `${normalizeDocPath(normalized)}:${lineMatch[1]}`;
      return normalized;
    }
    if (headingMatch && headingMatch[1]) {
      normalized = `${normalizeDocPath(normalized)}#${headingMatch[1]}`;
      return normalized;
    }
  }

  return normalizeDocPath(normalized);
}

export function getInlineMention(raw: string): { type: "task"; taskId: string } | { type: "doc"; docPath: string } | null {
  const value = raw.trim();

  // Skip placeholder/example text containing angle brackets (e.g. @doc/<path>)
  if (value.includes("<") || value.includes(">")) return null;

  if (/^@task-[a-zA-Z0-9]+(?:\.[a-zA-Z0-9]+)?$/.test(value)) {
    return { type: "task", taskId: value.slice(1) };
  }

  if (
    value.startsWith("@doc/") ||
    value.startsWith("@docs/") ||
    value.startsWith(".knowns/docs/") ||
    value.startsWith("/.knowns/docs/")
  ) {
    return { type: "doc", docPath: toDocPath(value) };
  }

  return null;
}

/**
 * Transform mention patterns into markdown links
 * These will then be styled via the custom link component
 * IMPORTANT: Skip code blocks to avoid breaking mermaid/code syntax
 */
export function transformMentions(content: string): string {
  const parts: string[] = [];
  let lastIndex = 0;

  const codeBlockRegex = /(```[\s\S]*?```|`[^`\n]+`)/g;
  let match: RegExpExecArray | null;

  while ((match = codeBlockRegex.exec(content)) !== null) {
    if (match.index > lastIndex) {
      parts.push(transformMentionsInText(content.slice(lastIndex, match.index)));
    }
    parts.push(match[0]);
    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < content.length) {
    parts.push(transformMentionsInText(content.slice(lastIndex)));
  }

  return parts.join("");
}

/**
 * Transform mentions in regular text (not code)
 */
function transformMentionsInText(text: string): string {
  let transformed = normalizeKnownsTaskReferences(text);

  // Transform @task-123 to [@@task-123](/tasks/task-123)
  // Skip placeholder/example IDs with angle brackets (e.g. @task-<id>)
  transformed = transformed.replace(TASK_MENTION_REGEX, (_match, taskRef) => {
    if (taskRef.includes("<") || taskRef.includes(">")) return _match;
    return `[@@${taskRef}](/tasks/${taskRef})`;
  });

  // Transform @doc/path or @docs/path to [@@doc/path.md](/docs/path.md)
  // Supports @doc/path:line, @doc/path:start-end, @doc/path#heading
  transformed = transformed.replace(DOC_MENTION_REGEX, (_match, docPath) => {
    // Skip placeholder/example paths with angle brackets (e.g. <path>)
    if (docPath.includes("<") || docPath.includes(">")) return _match;

    let cleanPath = docPath;
    cleanPath = cleanPath.replace(/[:]+$/, "");
    if (cleanPath.endsWith(".") && !cleanPath.match(/\.\w+$/)) {
      cleanPath = cleanPath.slice(0, -1);
    }
    let fragment = "";
    let displaySuffix = "";
    const rangeMatch = cleanPath.match(/:(\d+)-(\d+)$/);
    const lineMatch = !rangeMatch && cleanPath.match(/:(\d+)$/);
    const headingMatch = !rangeMatch && !lineMatch && cleanPath.match(/#([a-zA-Z0-9_-]+(?:[a-zA-Z0-9_. -]*)?)$/);

    if (rangeMatch) {
      fragment = `?L=${rangeMatch[1]}-${rangeMatch[2]}`;
      displaySuffix = `:${rangeMatch[1]}-${rangeMatch[2]}`;
      cleanPath = cleanPath.slice(0, -rangeMatch[0].length);
    } else if (lineMatch) {
      fragment = `?L=${lineMatch[1]}`;
      displaySuffix = `:${lineMatch[1]}`;
      cleanPath = cleanPath.slice(0, -lineMatch[0].length);
    } else if (headingMatch) {
      fragment = `#${headingMatch[1]}`;
      displaySuffix = `#${headingMatch[1]}`;
      cleanPath = cleanPath.slice(0, -headingMatch[0].length);
    }
    const normalizedPath = toDocPath(cleanPath);
    return `[@@doc/${normalizedPath}${displaySuffix}](/docs/${normalizedPath}${fragment})`;
  });

  transformed = transformed.replace(KNOWNS_DOC_PATH_REGEX, (_match, prefix, docPath) => {
    const normalizedPath = toDocPath(docPath);
    return `${prefix}[@@doc/${normalizedPath}](/docs/${normalizedPath})`;
  });

  return transformed;
}

// Notion-like mention styles
const mentionBase =
  "inline-flex items-center gap-1 px-1 py-px rounded text-[0.9em] font-medium cursor-pointer select-none transition-all no-underline";

export const taskMentionClass =
  `${mentionBase} bg-green-500/8 text-green-700 hover:bg-green-500/15 hover:underline decoration-green-500/40 dark:text-green-400`;

export const taskMentionBrokenClass =
  `${mentionBase} bg-red-500/8 text-red-600 line-through opacity-70 cursor-not-allowed dark:text-red-400`;

export const docMentionClass =
  `${mentionBase} bg-blue-500/8 text-blue-700 hover:bg-blue-500/15 hover:underline decoration-blue-500/40 dark:text-blue-400`;

export const docMentionBrokenClass =
  `${mentionBase} bg-red-500/8 text-red-600 line-through opacity-70 cursor-not-allowed dark:text-red-400`;

// Status colors for task badges
export const STATUS_STYLES: Record<string, string> = {
  todo: "bg-muted-foreground/50",
  "in-progress": "bg-yellow-500",
  "in-review": "bg-purple-500",
  blocked: "bg-red-500",
  done: "bg-green-500",
};
