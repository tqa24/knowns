import { cn } from "@/ui/lib/utils";
import type { FilterState } from "./constants";

import {
	CODE_EDGE_ORDER,
	CODE_KIND_ORDER,
	codeEdgeColors,
	codeEdgeFilterKey,
	codeEdgeLabels,
	codeKindFilterKey,
	countCodeEdges,
	countCodeKinds,
	getCodeKindColor,
	getCodeKindLabel,
	type CodeFilterState,
} from "./constants";
import type { GraphData } from "@/ui/api/client";

interface GraphLegendProps {
	data: GraphData | null;
	filters: FilterState;
	onToggleFilter: (key: keyof FilterState) => void;
}

function countKnowledgeNodeTypes(data: GraphData | null) {
	const counts = { task: 0, doc: 0, memory: 0 };
	if (!data) return counts;
	for (const node of data.nodes) {
		if (node.type === "task" || node.type === "doc" || node.type === "memory") counts[node.type] += 1;
	}
	return counts;
}

function countKnowledgeEdgeTypes(data: GraphData | null) {
	const counts = { parent: 0, spec: 0, mention: 0 };
	if (!data) return counts;
	for (const edge of data.edges) {
		if (edge.type === "parent" || edge.type === "spec" || edge.type === "mention") counts[edge.type] += 1;
	}
	return counts;
}

export function GraphLegend({ data, filters, onToggleFilter }: GraphLegendProps) {
	const nodeCounts = countKnowledgeNodeTypes(data);
	const edgeCounts = countKnowledgeEdgeTypes(data);
	return (
		<div className="absolute bottom-3 left-3 z-10 flex flex-col gap-1.5 rounded-lg border bg-background/90 backdrop-blur-sm p-2.5 text-xs shadow-sm">
			<div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Node types</div>
			<div className="flex flex-col gap-1">
				{[
					{ key: "tasks" as const, label: "Tasks", color: "#6366f1", active: filters.tasks, count: nodeCounts.task },
					{ key: "docs" as const, label: "Docs", color: "#f59e0b", active: filters.docs, count: nodeCounts.doc },
					{ key: "memories" as const, label: "Memories", color: "#22c55e", active: filters.memories, count: nodeCounts.memory },
				].map((item) => (
					<button
						key={item.key}
						type="button"
						onClick={() => onToggleFilter(item.key)}
						className={cn(
							"flex items-center justify-between gap-3 rounded-md border px-2 py-1 text-left transition-colors",
							item.active ? "border-border bg-background/80" : "border-transparent opacity-45 hover:opacity-80",
						)}
					>
						<span className="flex items-center gap-1.5">
							<span className="h-3 w-3 rounded-full" style={{ backgroundColor: item.color }} />
							<span className="text-muted-foreground">{item.label}</span>
						</span>
						<span className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">{item.count}</span>
					</button>
				))}
			</div>
			<div className="h-px bg-border my-0.5" />
			<div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Edges</div>
			<div className="flex flex-col gap-1">
				{[
					{ key: "edgeParent" as const, label: "Parent", active: filters.edgeParent, count: edgeCounts.parent, className: "border-gray-400" },
					{ key: "edgeSpec" as const, label: "Spec", active: filters.edgeSpec, count: edgeCounts.spec, className: "border-indigo-500" },
					{ key: "edgeMention" as const, label: "Mention", active: filters.edgeMention, count: edgeCounts.mention, className: "border-gray-400 border-dotted" },
				].map((item) => (
					<button
						key={item.key}
						type="button"
						onClick={() => onToggleFilter(item.key)}
						className={cn(
							"flex items-center justify-between gap-3 rounded-md border px-2 py-1 text-left transition-colors",
							item.active ? "border-border bg-background/80" : "border-transparent opacity-45 hover:opacity-80",
						)}
					>
						<span className="flex items-center gap-1.5">
							<span className={cn("w-4 border-t-2", item.className)} />
							<span className="text-muted-foreground">{item.label}</span>
						</span>
						<span className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">{item.count}</span>
					</button>
				))}
			</div>
		</div>
	);
}

interface CodeGraphLegendProps {
	data: GraphData | null;
	filters: CodeFilterState;
	onToggleKind: (key: keyof CodeFilterState) => void;
}

export function CodeGraphLegend({ data, filters, onToggleKind }: CodeGraphLegendProps) {
	const counts = countCodeKinds(data);
	const edgeCounts = countCodeEdges(data);

	return (
		<div className="absolute bottom-3 left-3 z-10 flex flex-col gap-1.5 rounded-lg border bg-background/90 backdrop-blur-sm p-2.5 text-xs shadow-sm">
			<div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Node types</div>
			<div className="flex flex-col gap-1">
				{CODE_KIND_ORDER.map((kind) => {
					const filterKey = codeKindFilterKey(kind);
					const active = filters[filterKey];
					return (
						<button
							key={kind}
							type="button"
							onClick={() => onToggleKind(filterKey)}
							className={cn(
								"flex items-center justify-between gap-3 rounded-md border px-2 py-1 text-left transition-colors",
								active ? "border-border bg-background/80" : "border-transparent opacity-45 hover:opacity-80",
							)}
						>
							<span className="flex items-center gap-1.5">
								<span className="h-3 w-3 rounded-full" style={{ backgroundColor: getCodeKindColor(kind) }} />
								<span className="text-muted-foreground">{getCodeKindLabel(kind)}</span>
							</span>
							<span className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">{counts[kind]}</span>
						</button>
					);
				})}
			</div>
			<div className="h-px bg-border my-0.5" />
			<div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Edges</div>
			<div className="flex flex-col gap-1">
				{CODE_EDGE_ORDER.map((kind) => {
					const filterKey = codeEdgeFilterKey(kind);
					const active = filters[filterKey];
					return (
						<button
							key={kind}
							type="button"
							onClick={() => onToggleKind(filterKey)}
							className={cn(
								"flex items-center justify-between gap-3 rounded-md border px-2 py-1 text-left transition-colors",
								active ? "border-border bg-background/80" : "border-transparent opacity-45 hover:opacity-80",
							)}
						>
							<span className="flex items-center gap-1.5">
								<span className="w-4 border-t-2" style={{ borderColor: codeEdgeColors[kind] }} />
								<span className="text-muted-foreground">{codeEdgeLabels[kind]}</span>
							</span>
							<span className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">{edgeCounts[kind]}</span>
						</button>
					);
				})}
			</div>
		</div>
	);
}
