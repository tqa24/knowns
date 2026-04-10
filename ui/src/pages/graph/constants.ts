import type { GraphData, GraphEdge, GraphNode } from "@/ui/api/client";
import type cytoscape from "cytoscape";

export const TASK_COLOR = "#6366f1";
export const DOC_COLOR = "#f59e0b";
export const CODE_COLOR = "#a855f7";
export const MEMORY_COLOR = "#22c55e";

export const CODE_KIND_ORDER = ["file", "function", "method", "class", "interface"] as const;
export type CodeKind = (typeof CODE_KIND_ORDER)[number];

export const codeKindLabels: Record<CodeKind, string> = {
	file: "File",
	function: "Function",
	method: "Method",
	class: "Class",
	interface: "Interface",
};

export const codeKindColors: Record<CodeKind, string> = {
	file: "#64748b",
	function: "#3b82f6",
	method: "#06b6d4",
	class: "#a855f7",
	interface: "#22c55e",
};

export const statusBorderColors: Record<string, string> = {
	todo: "#6b7280",
	"in-progress": "#f59e0b",
	"in-review": "#a855f7",
	done: "#22c55e",
	blocked: "#ef4444",
	"on-hold": "#8b5cf6",
	urgent: "#dc2626",
};

export const memoryLayerColors: Record<string, string> = {
	working: "#6b7280",
	project: "#22c55e",
	global: "#a855f7",
};

export const CLUSTER_PALETTE = ["#ef4444", "#f59e0b", "#22c55e", "#3b82f6", "#8b5cf6", "#ec4899", "#14b8a6", "#f97316"];
export const LARGE_GRAPH_NODE_THRESHOLD = 2000;
export const LARGE_GRAPH_EDGE_THRESHOLD = 5000;

// FilterState for knowledge graph (tasks/docs/memories)
export interface FilterState {
	tasks: boolean;
	docs: boolean;
	memories: boolean;
	showIsolated: boolean;
	showEdges: boolean;
	edgeParent: boolean;
	edgeSpec: boolean;
	edgeMention: boolean;
}

export const KNOWLEDGE_FILTERS: FilterState = {
	tasks: true,
	docs: true,
	memories: true,
	showIsolated: true,
	showEdges: true,
	edgeParent: true,
	edgeSpec: true,
	edgeMention: true,
};

// CodeFilterState for code graph page
export interface CodeFilterState {
	showIsolated: boolean;
	showEdges: boolean;
	edgeCalls: boolean;
	edgeHasMethod: boolean;
	edgeImports: boolean;
	edgeContains: boolean;
	edgeExtends: boolean;
	edgeImplements: boolean;
	edgeInstantiates: boolean;
	kindFile: boolean;
	kindFunction: boolean;
	kindMethod: boolean;
	kindClass: boolean;
	kindInterface: boolean;
}

export const CODE_GRAPH_FILTERS: CodeFilterState = {
	showIsolated: true,
	showEdges: true,
	edgeCalls: true,
	edgeHasMethod: true,
	edgeImports: true,
	edgeContains: true,
	edgeExtends: true,
	edgeImplements: true,
	edgeInstantiates: true,
	kindFile: true,
	kindFunction: true,
	kindMethod: true,
	kindClass: true,
	kindInterface: true,
};

export function codeKindFilterKey(kind: CodeKind): keyof CodeFilterState {
	switch (kind) {
	case "file":
		return "kindFile";
	case "function":
		return "kindFunction";
	case "method":
		return "kindMethod";
	case "class":
		return "kindClass";
	case "interface":
		return "kindInterface";
	}
}

export function getCodeNodeKind(node: GraphNode): CodeKind {
	const kind = typeof node.data.kind === "string" ? node.data.kind.toLowerCase() : "";
	switch (kind) {
	case "file":
	case "function":
	case "method":
	case "class":
	case "interface":
		return kind;
	default:
		return "function";
	}
}

export function getVisibleCodeKinds(filters: CodeFilterState): Set<CodeKind> {
	const visibleKinds = new Set<CodeKind>();
	if (filters.kindFile) visibleKinds.add("file");
	if (filters.kindFunction) visibleKinds.add("function");
	if (filters.kindMethod) visibleKinds.add("method");
	if (filters.kindClass) visibleKinds.add("class");
	if (filters.kindInterface) visibleKinds.add("interface");
	return visibleKinds;
}

export function countCodeKinds(data: GraphData | null): Record<CodeKind, number> {
	const counts: Record<CodeKind, number> = {
		file: 0,
		function: 0,
		method: 0,
		class: 0,
		interface: 0,
	};
	if (!data) return counts;
	for (const node of data.nodes) {
		counts[getCodeNodeKind(node)] += 1;
	}
	return counts;
}

export const CODE_EDGE_ORDER = ["calls", "has_method", "imports", "contains", "extends", "implements", "instantiates"] as const;
export type CodeEdgeKind = (typeof CODE_EDGE_ORDER)[number];

export const codeEdgeLabels: Record<CodeEdgeKind, string> = {
	calls: "Calls",
	has_method: "Has Method",
	imports: "Imports",
	contains: "Contains",
	extends: "Extends",
	implements: "Implements",
	instantiates: "Creates",
};

export const codeEdgeColors: Record<CodeEdgeKind, string> = {
	calls: "#f97316",
	has_method: "#ec4899",
	imports: "#14b8a6",
	contains: "#94a3b8",
	extends: "#8b5cf6",
	implements: "#6366f1",
	instantiates: "#eab308",
};

export function codeEdgeFilterKey(kind: CodeEdgeKind): keyof CodeFilterState {
	switch (kind) {
	case "calls":
		return "edgeCalls";
	case "has_method":
		return "edgeHasMethod";
	case "imports":
		return "edgeImports";
	case "contains":
		return "edgeContains";
	case "extends":
		return "edgeExtends";
	case "implements":
		return "edgeImplements";
	case "instantiates":
		return "edgeInstantiates";
	}
}

export function getVisibleCodeEdgeKinds(filters: CodeFilterState): Set<CodeEdgeKind> {
	const visible = new Set<CodeEdgeKind>();
	if (filters.edgeCalls) visible.add("calls");
	if (filters.edgeHasMethod) visible.add("has_method");
	if (filters.edgeImports) visible.add("imports");
	if (filters.edgeContains) visible.add("contains");
	if (filters.edgeExtends) visible.add("extends");
	if (filters.edgeImplements) visible.add("implements");
	if (filters.edgeInstantiates) visible.add("instantiates");
	return visible;
}

export function countCodeEdges(data: GraphData | null): Record<CodeEdgeKind, number> {
	const counts: Record<CodeEdgeKind, number> = {
		calls: 0,
		has_method: 0,
		imports: 0,
		contains: 0,
		extends: 0,
		implements: 0,
		instantiates: 0,
	};
	if (!data) return counts;
	for (const edge of data.edges) {
		if ((CODE_EDGE_ORDER as readonly string[]).includes(edge.type)) {
			counts[edge.type as CodeEdgeKind] += 1;
		}
	}
	return counts;
}

export function getCodeKindColor(kind: CodeKind): string {
	return codeKindColors[kind];
}

export function getCodeKindLabel(kind: CodeKind): string {
	return codeKindLabels[kind];
}

export interface GraphReferenceItem {
	nodeId: string;
	label: string;
	type: GraphNode["type"] | "external";
	edgeType: GraphEdge["type"];
	isVirtual?: boolean;
	resolutionStatus?: string;
}

export interface SelectedNodeReferences {
	incoming: GraphReferenceItem[];
	outgoing: GraphReferenceItem[];
}

export function buildElements(data: GraphData, filters: FilterState): cytoscape.ElementDefinition[] {
	const visibleTypes = new Set<string>();
	if (filters.tasks) visibleTypes.add("task");
	if (filters.docs) visibleTypes.add("doc");
	if (filters.memories) visibleTypes.add("memory");

	const typeFilteredIds = new Set(data.nodes.filter((n) => visibleTypes.has(n.type)).map((n) => n.id));

	const visibleEdgeTypes = new Set<string>();
	if (filters.edgeParent) visibleEdgeTypes.add("parent");
	if (filters.edgeSpec) visibleEdgeTypes.add("spec");
	if (filters.edgeMention) visibleEdgeTypes.add("mention");

	const validEdges = filters.showEdges
		? data.edges.filter(
				(e) => typeFilteredIds.has(e.source) && typeFilteredIds.has(e.target) && visibleEdgeTypes.has(e.type),
			)
		: [];

	const connectedIds = new Set<string>();
	for (const e of validEdges) {
		connectedIds.add(e.source);
		connectedIds.add(e.target);
	}

	const visibleNodeIds = new Set([...typeFilteredIds].filter((id) => filters.showIsolated || connectedIds.has(id)));

	const nodes: cytoscape.ElementDefinition[] = data.nodes
		.filter((n) => visibleNodeIds.has(n.id))
		.map((n) => {
			const color =
				n.type === "task"
					? TASK_COLOR
					: n.type === "code"
						? CODE_COLOR
						: n.type === "memory"
							? memoryLayerColors[(n.data.layer as string) || "project"] || MEMORY_COLOR
							: DOC_COLOR;
			const borderColor =
				n.type === "task"
					? statusBorderColors[(n.data.status as string) || "todo"] || "#6b7280"
					: "transparent";
			return {
				data: {
					...n.data,
					id: n.id,
					label: n.label,
					nodeType: n.type,
					color,
					borderColor,
					status: n.data.status,
					priority: n.data.priority,
				},
			};
		});

	const edges: cytoscape.ElementDefinition[] = validEdges
		.filter((e) => visibleNodeIds.has(e.source) && visibleNodeIds.has(e.target))
		.map((e) => ({
			data: {
				id: `${e.source}-${e.type}-${e.target}`,
				source: e.source,
				target: e.target,
				edgeType: e.type,
			},
		}));

	return [...nodes, ...edges];
}

export function buildOverviewGraphElements(data: GraphData, maxNodes = 300): cytoscape.ElementDefinition[] {
	if (!data) return [];
	const degree = new Map<string, number>();
	for (const node of data.nodes) degree.set(node.id, 0);
	for (const edge of data.edges) {
		degree.set(edge.source, (degree.get(edge.source) ?? 0) + 1);
		degree.set(edge.target, (degree.get(edge.target) ?? 0) + 1);
	}
	const topNodes = [...data.nodes]
		.sort((a, b) => (degree.get(b.id) ?? 0) - (degree.get(a.id) ?? 0))
		.slice(0, maxNodes);
	const allowed = new Set(topNodes.map((n) => n.id));
	const overviewData: GraphData = {
		nodes: topNodes,
		edges: data.edges.filter((e) => allowed.has(e.source) && allowed.has(e.target)),
	};
	return buildElements(overviewData, KNOWLEDGE_FILTERS);
}

export function buildOverviewCodeGraphElements(data: GraphData, maxNodes = 400): cytoscape.ElementDefinition[] {
	if (!data) return [];
	const degree = new Map<string, number>();
	for (const node of data.nodes) degree.set(node.id, 0);
	for (const edge of data.edges) {
		degree.set(edge.source, (degree.get(edge.source) ?? 0) + 1);
		degree.set(edge.target, (degree.get(edge.target) ?? 0) + 1);
	}
	const topNodes = [...data.nodes]
		.sort((a, b) => {
			const degreeDiff = (degree.get(b.id) ?? 0) - (degree.get(a.id) ?? 0);
			if (degreeDiff !== 0) return degreeDiff;
			const aKind = getCodeNodeKind(a);
			const bKind = getCodeNodeKind(b);
			return CODE_KIND_ORDER.indexOf(bKind) - CODE_KIND_ORDER.indexOf(aKind);
		})
		.slice(0, maxNodes);
	const allowed = new Set(topNodes.map((n) => n.id));
	const overviewData: GraphData = {
		nodes: topNodes,
		edges: data.edges.filter((e) => allowed.has(e.source) && allowed.has(e.target)),
	};
	return buildCodeElements(overviewData, CODE_GRAPH_FILTERS);
}

export function buildCodeElements(data: GraphData, filters: CodeFilterState): cytoscape.ElementDefinition[] {
	const visibleKinds = getVisibleCodeKinds(filters);
	const typeFilteredIds = new Set(
		data.nodes
			.filter((n) => visibleKinds.has(getCodeNodeKind(n)))
			.map((n) => n.id),
	);

	const visibleEdgeTypes = getVisibleCodeEdgeKinds(filters);

	const validEdges = filters.showEdges
		? data.edges.filter(
				(e) => typeFilteredIds.has(e.source) && typeFilteredIds.has(e.target) && visibleEdgeTypes.has(e.type),
			)
		: [];

	const connectedIds = new Set<string>();
	for (const e of validEdges) {
		connectedIds.add(e.source);
		connectedIds.add(e.target);
	}

	const visibleNodeIds = new Set([...typeFilteredIds].filter((id) => filters.showIsolated || connectedIds.has(id)));

	const nodes: cytoscape.ElementDefinition[] = data.nodes
		.filter((n) => visibleNodeIds.has(n.id))
		.map((n) => {
			const codeKind = getCodeNodeKind(n);
			return {
				data: {
					...n.data,
					id: n.id,
					label: n.label,
					nodeType: "code",
					codeKind,
					color: getCodeKindColor(codeKind),
					borderColor: "transparent",
				},
			};
		});

	const edges: cytoscape.ElementDefinition[] = validEdges
		.filter((e) => visibleNodeIds.has(e.source) && visibleNodeIds.has(e.target))
		.map((e) => ({
			data: {
				id: `${e.source}-${e.type}-${e.target}`,
				source: e.source,
				target: e.target,
				edgeType: e.type,
			},
		}));

	return [...nodes, ...edges];
}

export function buildSelectedNodeReferences(data: GraphData | null, selectedNode: GraphNode | null): SelectedNodeReferences {
	if (!data || !selectedNode) {
		return { incoming: [], outgoing: [] };
	}

	const dedupeRefs = (items: GraphReferenceItem[]) => {
		const seen = new Set<string>();
		return items.filter((item) => {
			const key = [item.nodeId, item.label, item.type, item.edgeType, item.resolutionStatus || "", item.isVirtual ? "1" : "0"].join("|");
			if (seen.has(key)) return false;
			seen.add(key);
			return true;
		});
	};

	const nodeMap = new Map(data.nodes.map((node) => [node.id, node] as const));
	const toRef = (edge: GraphEdge, relatedId: string) => {
		const relatedNode = nodeMap.get(relatedId);
		if (relatedNode) {
			return {
				nodeId: relatedNode.id,
				label: relatedNode.label,
				type: relatedNode.type,
				edgeType: edge.type,
			};
		}
		const displayTarget = typeof edge.data?.display_target === "string" ? edge.data.display_target : null;
		if (!displayTarget) return null;
		return {
			nodeId: relatedId,
			label: displayTarget,
			type: "external" as const,
			edgeType: edge.type,
			isVirtual: true,
			resolutionStatus: typeof edge.data?.resolution_status === "string" ? edge.data.resolution_status : undefined,
		};
	};

	const incoming = dedupeRefs(
		data.edges
		.filter((edge) => edge.target === selectedNode.id)
		.map((edge) => toRef(edge, edge.source))
		.filter((item): item is NonNullable<typeof item> => item !== null)
		.sort((a, b) => a.label.localeCompare(b.label)),
	);

	const outgoing = dedupeRefs(
		data.edges
		.filter((edge) => edge.source === selectedNode.id)
		.map((edge) => toRef(edge, edge.target))
		.filter((item): item is NonNullable<typeof item> => item !== null)
		.sort((a, b) => a.label.localeCompare(b.label)),
	);

	return { incoming, outgoing };
}
