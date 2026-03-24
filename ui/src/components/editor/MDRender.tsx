import {
  forwardRef,
  useImperativeHandle,
  useRef,
  useMemo,
  useState,
  lazy,
  Suspense,
  type ReactNode,
} from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import hljs from "highlight.js";
import { ClipboardCheck, Check, Loader2 } from "lucide-react";
import { useTheme } from "../../App";
import { cn } from "../../lib/utils";

import { transformMentions, toDocPath, getInlineMention } from "./mentionUtils";
import { TaskMentionBadge } from "./TaskMentionBadge";
import { DocMentionBadge } from "./DocMentionBadge";
import { MarkdownErrorBoundary } from "./MarkdownErrorBoundary";
import { extractTextFromChildren, slugifyHeading, parseHeadingMeta } from "./headingUtils";

// Lazy load MermaidBlock for better performance
const MermaidBlock = lazy(() => import("./MermaidBlock"));

function MermaidLoading() {
  return (
    <div className="my-4 p-4 rounded-lg border bg-muted/30 animate-pulse">
      <div className="h-32 flex items-center justify-center text-muted-foreground gap-2">
        <Loader2 className="w-4 h-4 animate-spin" />
        Loading diagram...
      </div>
    </div>
  );
}

export interface MDRenderRef {
  getElement: () => HTMLElement | null;
}

interface MDRenderProps {
  markdown: string;
  className?: string;
  onDocLinkClick?: (path: string) => void;
  onTaskLinkClick?: (taskId: string) => void;
  showHeadingAnchors?: boolean;
  onHeadingAnchorClick?: (id: string) => void;
}

/**
 * Read-only markdown renderer with mention badge support
 */
const MDRender = forwardRef<MDRenderRef, MDRenderProps>(
  ({ markdown, className = "", onDocLinkClick, onTaskLinkClick, showHeadingAnchors = false, onHeadingAnchorClick }, ref) => {
    const { isDark } = useTheme();
    const containerRef = useRef<HTMLDivElement>(null);

    const transformedMarkdown = useMemo(() => transformMentions(markdown || ""), [markdown]);

    useImperativeHandle(ref, () => ({
      getElement: () => containerRef.current,
    }));

    const headingMeta = useMemo(() => parseHeadingMeta(markdown || ""), [markdown]);
    let headingRenderIndex = 0;

    const Heading = ({ level, children, ...props }: { level: number; children?: ReactNode }) => {
      const Tag = `h${level}` as const;
      if (!showHeadingAnchors) {
        return <Tag {...props}>{children}</Tag>;
      }

      const text = extractTextFromChildren(children);
      const meta = headingMeta[headingRenderIndex];
      headingRenderIndex += 1;

      const number = meta?.number ?? "";
      const id = meta?.id ?? slugifyHeading(text);

      const headingClassName =
        level <= 3
          ? "group relative scroll-mt-4 flex items-baseline gap-2"
          : "group relative scroll-mt-4 flex items-baseline gap-1.5";

      return (
        <Tag id={id} className={headingClassName} {...props}>
          {number && (
            <span className="inline-block text-[0.72em] font-semibold text-muted-foreground/50 select-none leading-none translate-y-[1px]">
              {number}
            </span>
          )}
          <span className="min-w-0">{children}</span>
          <a
            href={`#${id}`}
            className="ml-1 text-muted-foreground/0 group-hover:text-muted-foreground/50 hover:!text-foreground transition-colors no-underline leading-none"
            aria-label={`Link to section ${number}${text ? ` ${text}` : ""}`}
            onClick={(e) => {
              if (onHeadingAnchorClick) {
                e.preventDefault();
                onHeadingAnchorClick(id);
              }
            }}
          >
            #
          </a>
        </Tag>
      );
    };

    const components = useMemo(
      () => ({
        h2: ({ children, ...props }: { children?: ReactNode }) => (
          <Heading level={2} {...props}>{children}</Heading>
        ),
        h3: ({ children, ...props }: { children?: ReactNode }) => (
          <Heading level={3} {...props}>{children}</Heading>
        ),
        h4: ({ children, ...props }: { children?: ReactNode }) => (
          <Heading level={4} {...props}>{children}</Heading>
        ),

        a: ({ href, children }: { href?: string; children?: ReactNode }) => {
          const text = String(children);

          if (text.startsWith("@@task-")) {
            return <TaskMentionBadge taskId={text.slice(2)} onTaskLinkClick={onTaskLinkClick} />;
          }

          if (text.startsWith("@@doc/")) {
            return <DocMentionBadge docPath={text.slice(6)} onDocLinkClick={onDocLinkClick} />;
          }

          if (href && (href.startsWith("@doc/") || href.startsWith("@docs/") || href.startsWith(".knowns/docs/") || href.startsWith("/.knowns/docs/"))) {
            return <DocMentionBadge docPath={toDocPath(href)} onDocLinkClick={onDocLinkClick} />;
          }

          return <a href={href} className="text-primary hover:underline">{children}</a>;
        },

        code: ({
          className: codeClassName,
          children,
          node,
          ...props
        }: {
          className?: string;
          children?: ReactNode;
          node?: unknown;
        }) => {
          const match = /language-(\w+)/.exec(codeClassName || "");
          const language = match?.[1];
          const codeContent = String(children).replace(/\n$/, "");
          const isInline = !language && !String(children).includes("\n");

          if (isInline) {
            const inlineMention = getInlineMention(codeContent);
            if (inlineMention?.type === "task") {
              return <TaskMentionBadge taskId={inlineMention.taskId} onTaskLinkClick={onTaskLinkClick} />;
            }
            if (inlineMention?.type === "doc") {
              return <DocMentionBadge docPath={inlineMention.docPath} onDocLinkClick={onDocLinkClick} />;
            }
            return (
              <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-sm whitespace-break-spaces break-words [overflow-wrap:anywhere]" {...props}>
                {children}
              </code>
            );
          }

          if (language === "mermaid") {
            const key = `mermaid-${codeContent.length}-${codeContent.slice(0, 50).replace(/\s/g, '')}`;
            return (
              <Suspense key={key} fallback={<MermaidLoading />}>
                <MermaidBlock code={codeContent} />
              </Suspense>
            );
          }

          let highlightedCode: string;
          try {
            if (language && hljs.getLanguage(language)) {
              highlightedCode = hljs.highlight(codeContent, { language }).value;
            } else {
              highlightedCode = hljs.highlightAuto(codeContent).value;
            }
          } catch {
            highlightedCode = codeContent;
          }

          return (
            <code
              className={`hljs ${language ? `language-${language}` : ""}`}
              dangerouslySetInnerHTML={{ __html: highlightedCode }}
            />
          );
        },

        pre: ({ children, ...props }: { children?: ReactNode }) => {
          const [copied, setCopied] = useState(false);
          const preRef = useRef<HTMLPreElement>(null);

          const handleCopy = () => {
            const text = preRef.current?.textContent || "";
            navigator.clipboard.writeText(text).then(() => {
              setCopied(true);
              setTimeout(() => setCopied(false), 2000);
            });
          };

          return (
            <div className="group relative">
              <button
                type="button"
                onClick={handleCopy}
                className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-md bg-muted hover:bg-muted/80 border border-border text-muted-foreground hover:text-foreground"
                title="Copy code"
              >
                {copied ? <Check className="w-3.5 h-3.5 text-green-500" /> : <ClipboardCheck className="w-3.5 h-3.5" />}
              </button>
              <pre ref={preRef} className="p-4 rounded-lg overflow-x-auto text-sm hljs-pre" {...props}>
                {children}
              </pre>
            </div>
          );
        },

        input: ({ type, checked, disabled, ...props }: { type?: string; checked?: boolean; disabled?: boolean }) => {
          if (type === "checkbox") {
            return (
              <span
                className={cn(
                  "inline-flex items-center justify-center h-4 w-4 shrink-0 rounded-sm border mr-2 align-text-bottom",
                  checked ? "bg-primary border-primary text-primary-foreground" : "border-muted-foreground/50"
                )}
                aria-checked={checked}
                role="checkbox"
              >
                {checked && <Check className="h-3 w-3" />}
              </span>
            );
          }
          return <input type={type} checked={checked} disabled={disabled} {...props} />;
        },

        li: ({ children, className, ...props }: { children?: ReactNode; className?: string }) => {
          const isTaskListItem = className?.includes("task-list-item");
          return (
            <li className={cn(className, isTaskListItem && "list-none ml-0 flex items-start gap-0")} {...props}>
              {children}
            </li>
          );
        },

        ul: ({ children, className, ...props }: { children?: ReactNode; className?: string }) => {
          const isTaskList = className?.includes("contains-task-list");
          return (
            <ul className={cn(className, isTaskList && "list-none pl-0")} {...props}>
              {children}
            </ul>
          );
        },

        table: ({ children, ...props }: { children?: ReactNode }) => {
          const [copied, setCopied] = useState(false);
          const tableRef = useRef<HTMLTableElement>(null);

          const handleCopyTable = () => {
            if (!tableRef.current) return;
            const rows = tableRef.current.querySelectorAll("tr");
            const markdownRows: string[] = [];
            rows.forEach((row, rowIndex) => {
              const cells = row.querySelectorAll("th, td");
              const cellTexts = Array.from(cells).map((cell) => cell.textContent?.trim() || "");
              markdownRows.push(`| ${cellTexts.join(" | ")} |`);
              if (rowIndex === 0) {
                markdownRows.push(`| ${cellTexts.map(() => "---").join(" | ")} |`);
              }
            });
            navigator.clipboard.writeText(markdownRows.join("\n")).then(() => {
              setCopied(true);
              setTimeout(() => setCopied(false), 2000);
            });
          };

          return (
            <div className="table-wrapper group relative my-4 overflow-x-auto rounded-lg border border-border">
              <button
                type="button"
                onClick={handleCopyTable}
                className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-md bg-muted hover:bg-muted/80 border border-border text-muted-foreground hover:text-foreground"
                title="Copy as Markdown"
              >
                {copied ? <Check className="w-4 h-4 text-green-500" /> : <ClipboardCheck className="w-4 h-4" />}
              </button>
              <table ref={tableRef} className="w-full border-collapse text-sm" {...props}>
                {children}
              </table>
            </div>
          );
        },

        thead: ({ children, ...props }: { children?: ReactNode }) => (
          <thead className="bg-muted" {...props}>{children}</thead>
        ),
        tbody: ({ children, ...props }: { children?: ReactNode }) => (
          <tbody className="divide-y divide-border/50" {...props}>{children}</tbody>
        ),
        tr: ({ children, ...props }: { children?: ReactNode }) => (
          <tr className="hover:bg-muted/50 transition-colors" {...props}>{children}</tr>
        ),
        th: ({ children, ...props }: { children?: ReactNode }) => (
          <th className="px-4 py-3 text-left font-semibold border-b-2 border-border" {...props}>{children}</th>
        ),
        td: ({ children, ...props }: { children?: ReactNode }) => (
          <td className="px-4 py-3 border-b border-border/30" {...props}>{children}</td>
        ),
      }),
      [Heading, onDocLinkClick, onTaskLinkClick, isDark]
    );

    if (!markdown) return null;

    return (
      <div
        ref={containerRef}
        className={`md-render-wrapper ${className}`}
        data-color-mode={isDark ? "dark" : "light"}
      >
        <MarkdownErrorBoundary>
          <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
            {transformedMarkdown}
          </ReactMarkdown>
        </MarkdownErrorBoundary>
      </div>
    );
  },
);

MDRender.displayName = "MDRender";

export default MDRender;
