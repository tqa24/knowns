import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import ForceGraph2D, { type ForceGraphMethods } from "react-force-graph-2d";
import { Loader2, Maximize2, Minimize2, Scan, Search, X } from "lucide-react";

import { getCodeGraph, type GraphData, type GraphEdge, type GraphNode } from "@/ui/api/client";
import { useTheme } from "@/ui/App";

import { GraphDetailPanel } from "./GraphDetailPanel";
import { CodeGraphLegend } from "./graph/GraphLegend";
import { useContainerSize } from "./graph/useContainerSize";
import {
	buildOverviewCodeGraphElements,
	buildSelectedNodeReferences,
	CODE_GRAPH_FILTERS,
	countCodeKinds,
	getCodeKindColor,
	getCodeNodeKind,
	getVisibleCodeEdgeKinds,
	getVisibleCodeKinds,
	LARGE_GRAPH_EDGE_THRESHOLD,
	LARGE_GRAPH_NODE_THRESHOLD,
	type CodeFilterState,
} from "./graph/constants";

type ForceNode = GraphNode & {
	color: string;
	val?: number;
	x?: number;
	y?: number;
	vx?: number;
	vy?: number;
};

type ForceLink = GraphEdge & {
	id: string;
	source: string | ForceNode;
	target: string | ForceNode;
	color: string;
	width: number;
	dashed?: boolean;
};

const EMPTY_FORCE_DATA = { nodes: [], links: [], matches: 0 };

function lockForceNodes(nodes: ForceNode[]) {
	for (const node of nodes) {
		if (typeof node.x === "number") node.fx = node.x;
		if (typeof node.y === "number") node.fy = node.y;
	}
}

function unlockForceNodes(nodes: ForceNode[]) {
	for (const node of nodes) {
		delete node.fx;
		delete node.fy;
	}
}

function edgeId(edge: GraphEdge): string {
	return `${edge.source}-${edge.type}-${edge.target}`;
}

function edgeColor(type: GraphEdge["type"]): string {
	switch (type) {
		case "calls":
			return "#f97316";
		case "has_method":
			return "#ec4899";
		case "imports":
			return "#14b8a6";
		case "contains":
			return "#94a3b8";
		case "extends":
			return "#8b5cf6";
		case "implements":
			return "#6366f1";
		case "instantiates":
			return "#eab308";
		default:
			return "#94a3b8";
	}
}

function filterCodeGraphData(data: GraphData, filters: CodeFilterState): GraphData {
	const visibleKinds = getVisibleCodeKinds(filters);
	const visibleNodeIds = new Set(data.nodes.filter((n) => visibleKinds.has(getCodeNodeKind(n))).map((n) => n.id));
	const visibleEdgeKinds = getVisibleCodeEdgeKinds(filters);
	const edges = filters.showEdges
		? data.edges.filter((e) => visibleNodeIds.has(e.source) && visibleNodeIds.has(e.target) && visibleEdgeKinds.has(e.type as any))
		: [];
	const connectedIds = new Set<string>();
	for (const edge of edges) {
		connectedIds.add(edge.source);
		connectedIds.add(edge.target);
	}
	const nodes = data.nodes.filter((n) => visibleNodeIds.has(n.id) && (filters.showIsolated || connectedIds.has(n.id)));
	return { nodes, edges };
}

function toOverviewGraphData(data: GraphData): GraphData {
	const elements = buildOverviewCodeGraphElements(data);
	const nodeIds = new Set<string>();
	const edgeIds = new Set<string>();
	for (const el of elements) {
		const source = typeof el.data?.source === "string" ? el.data.source : null;
		const target = typeof el.data?.target === "string" ? el.data.target : null;
		if (source && target) {
			edgeIds.add(`${source}-${String(el.data?.edgeType)}-${target}`);
		} else if (typeof el.data?.id === "string") {
			nodeIds.add(el.data.id);
		}
	}
	return {
		nodes: data.nodes.filter((n) => nodeIds.has(n.id)),
		edges: data.edges.filter((e) => edgeIds.has(edgeId(e))),
	};
}

function computeNeighborhood(data: GraphData, rootId: string, hops: number) {
	const adjacency = new Map<string, Set<string>>();
	for (const node of data.nodes) adjacency.set(node.id, new Set());
	for (const edge of data.edges) {
		adjacency.get(edge.source)?.add(edge.target);
		adjacency.get(edge.target)?.add(edge.source);
	}
	const distances = new Map<string, number>();
	const queue: string[] = [rootId];
	distances.set(rootId, 0);
	while (queue.length > 0) {
		const current = queue.shift()!;
		const dist = distances.get(current)!;
		if (dist >= hops) continue;
		for (const next of adjacency.get(current) || []) {
			if (!distances.has(next)) {
				distances.set(next, dist + 1);
				queue.push(next);
			}
		}
	}
	return distances;
}

function buildForceData(data: GraphData, searchQuery: string, selectedNodeId: string | null): { nodes: ForceNode[]; links: ForceLink[]; matches: number } {
	const query = searchQuery.trim().toLowerCase();
	const matchedIds = new Set<string>();
	if (query) {
		for (const node of data.nodes) {
			const haystack = `${node.label} ${String(node.data.docPath || "")}`.toLowerCase();
			if (haystack.includes(query)) matchedIds.add(node.id);
		}
	}
	const activeIds = new Set<string>(matchedIds);
	if (selectedNodeId) {
		const distances = computeNeighborhood(data, selectedNodeId, 3);
		for (const id of distances.keys()) activeIds.add(id);
	} else if (matchedIds.size > 0) {
		for (const edge of data.edges) {
			if (matchedIds.has(edge.source) || matchedIds.has(edge.target)) {
				activeIds.add(edge.source);
				activeIds.add(edge.target);
			}
		}
	}

	const dimming = selectedNodeId || matchedIds.size > 0;

	const nodes: ForceNode[] = data.nodes.map((node) => ({
		...node,
		color: dimming && !activeIds.has(node.id) ? "rgba(148,163,184,0.25)" : getCodeKindColor(getCodeNodeKind(node)),
		val: getCodeNodeKind(node) === "file" ? 10 : getCodeNodeKind(node) === "class" || getCodeNodeKind(node) === "interface" ? 8 : 6,
	}));
	const links: ForceLink[] = data.edges.map((edge) => {
		const active = !dimming || (activeIds.has(edge.source) && activeIds.has(edge.target));
		return {
			...edge,
			id: edgeId(edge),
			color: active ? edgeColor(edge.type) : "rgba(148,163,184,0.15)",
			width: edge.type === "calls" || edge.type === "has_method" ? 2 : 1,
			dashed: edge.type === "imports" || edge.type === "implements" || edge.type === "extends",
		};
	});

	return { nodes, links, matches: matchedIds.size };
}

function sameNodeIds(a: ForceNode[], b: ForceNode[]): boolean {
	if (a.length !== b.length) return false;
	for (let i = 0; i < a.length; i++) {
		if (a[i].id !== b[i].id) return false;
	}
	return true;
}

function sameLinkIds(a: ForceLink[], b: ForceLink[]): boolean {
	if (a.length !== b.length) return false;
	for (let i = 0; i < a.length; i++) {
		if (a[i].id !== b[i].id) return false;
	}
	return true;
}

export default function CodeGraphPage() {
	const graphContainerRef = useRef<HTMLDivElement>(null);
	const graphRef = useRef<ForceGraphMethods<ForceNode, ForceLink> | undefined>(undefined);
	const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const stableForceDataRef = useRef<{ nodes: ForceNode[]; links: ForceLink[]; matches: number }>(EMPTY_FORCE_DATA);
	const { isDark } = useTheme();
	const { width, height } = useContainerSize(graphContainerRef);

	const [data, setData] = useState<GraphData | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [filters, setFilters] = useState<CodeFilterState>(CODE_GRAPH_FILTERS);
	const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
	const [isFullscreen, setIsFullscreen] = useState(false);
	const [searchQuery, setSearchQuery] = useState("");
	const [debouncedSearchQuery, setDebouncedSearchQuery] = useState("");
	const [engineRunning, setEngineRunning] = useState(false);

	const fetchCodeGraph = useCallback(async () => {
		setLoading(true);
		try {
			const graphData = await getCodeGraph();
			setData(graphData);
			setError(null);
		} catch (err) {
			setError("Failed to load code graph");
			console.error(err);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		fetchCodeGraph();
	}, [fetchCodeGraph]);

	useEffect(() => {
		if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
		searchTimerRef.current = setTimeout(() => setDebouncedSearchQuery(searchQuery), 200);
		return () => {
			if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
		};
	}, [searchQuery]);

	const isLargeGraph = !!data && (data.nodes.length >= LARGE_GRAPH_NODE_THRESHOLD || data.edges.length >= LARGE_GRAPH_EDGE_THRESHOLD);
	const filteredData = useMemo(() => {
		if (!data) return null;
		const base = filterCodeGraphData(data, filters);
		return isLargeGraph && !debouncedSearchQuery.trim() ? toOverviewGraphData(base) : base;
	}, [data, filters, isLargeGraph, debouncedSearchQuery]);

	const forceData = useMemo(() => {
		if (!filteredData) {
			stableForceDataRef.current = EMPTY_FORCE_DATA;
			return EMPTY_FORCE_DATA;
		}
		const next = buildForceData(filteredData, debouncedSearchQuery, selectedNode?.id ?? null);
		const prev = stableForceDataRef.current;
		const structureSame = sameNodeIds(prev.nodes, next.nodes) && sameLinkIds(prev.links, next.links);
		if (structureSame) {
			const merged = {
				nodes: prev.nodes.map((node, i) => ({ ...node, color: next.nodes[i].color, val: next.nodes[i].val })),
				links: prev.links.map((link, i) => ({ ...link, color: next.links[i].color, width: next.links[i].width, dashed: next.links[i].dashed })),
				matches: next.matches,
			};
			stableForceDataRef.current = merged;
			return merged;
		}
		stableForceDataRef.current = next;
		return next;
	}, [filteredData, debouncedSearchQuery, selectedNode?.id]);

	useEffect(() => {
		if (filteredData && (forceData.nodes.length > 0 || forceData.links.length > 0)) {
			setEngineRunning(true);
		} else {
			setEngineRunning(false);
		}
	}, [filteredData, forceData.nodes.length, forceData.links.length]);

	const toggleFilter = useCallback((key: keyof CodeFilterState) => {
		setFilters((prev) => ({ ...prev, [key]: !prev[key] }));
	}, []);

	const visibleKindCount = forceData.nodes.length;
	const edgeCount = forceData.links.length;
	const selectedNodeReferences = useMemo(() => buildSelectedNodeReferences(filteredData, selectedNode), [filteredData, selectedNode]);

	const handleZoomToFit = useCallback(() => {
		graphRef.current?.zoomToFit(400, 40);
	}, []);

	const clearSelection = useCallback(() => {
		setSelectedNode(null);
	}, []);

	const toGraphNode = useCallback((node: ForceNode): GraphNode => ({
		id: node.id,
		label: node.label,
		type: node.type,
		data: node.data,
	}), []);

	const toggleFullscreen = useCallback(() => {
		if (!document.fullscreenElement) {
			document.documentElement.requestFullscreen();
			setIsFullscreen(true);
		} else {
			document.exitFullscreen();
			setIsFullscreen(false);
		}
	}, []);

	if (error) {
		return (
			<div className="flex-1 flex items-center justify-center">
				<div className="text-center">
					<div className="text-destructive mb-2">{error}</div>
					<p className="text-xs text-muted-foreground">Run <code className="font-mono bg-muted px-1 rounded">knowns ingest</code> to index code files first.</p>
				</div>
			</div>
		);
	}

	return (
		<div className="flex-1 flex flex-col min-h-0 bg-background">
			<div className="flex items-center gap-2 px-4 py-2 border-b border-border/50 bg-background/95 flex-wrap">
				<div className="relative">
					<Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3 h-3 text-muted-foreground" />
					<input
						type="text"
						value={searchQuery}
						onChange={(e) => setSearchQuery(e.target.value)}
						placeholder="Search symbols..."
						className="h-7 w-40 rounded-md border bg-background pl-7 pr-7 text-xs placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
					/>
					{searchQuery && (
						<button
							type="button"
							onClick={() => setSearchQuery("")}
							className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
						>
							<X className="w-3 h-3" />
						</button>
					)}
				</div>

				{debouncedSearchQuery && <span className="text-xs text-muted-foreground">{forceData.matches} matches</span>}

				<div className="flex-1" />

				<span className="text-xs text-muted-foreground">
					{visibleKindCount} symbols, {edgeCount} edges
				</span>

				{engineRunning && <span className="text-xs text-amber-600 dark:text-amber-400">Layouting...</span>}

				<div className="flex items-center gap-0.5">
					<button
						type="button"
						onClick={handleZoomToFit}
						className="rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
						title="Zoom to fit"
					>
						<Scan className="w-4 h-4" />
					</button>
					<button
						type="button"
						onClick={toggleFullscreen}
						className="rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
						title="Toggle fullscreen"
					>
						{isFullscreen ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
					</button>
				</div>
			</div>

			<div className="flex-1 min-h-0 relative">
				<div ref={graphContainerRef} className="absolute inset-0" />
				{filteredData && (
					<ForceGraph2D
						ref={graphRef}
						width={width}
						height={height}
						graphData={{ nodes: forceData.nodes, links: forceData.links }}
						backgroundColor={isDark ? "#0b1020" : "#ffffff"}
						minZoom={0.1}
						maxZoom={8}
						enableZoomInteraction={true}
						enablePanInteraction={true}
						d3AlphaDecay={0.045}
						d3VelocityDecay={0.28}
						warmupTicks={0}
						cooldownTicks={isLargeGraph ? 80 : 140}
						cooldownTime={isLargeGraph ? 3000 : 4500}
						onEngineStop={() => {
							lockForceNodes(stableForceDataRef.current.nodes);
							setEngineRunning(false);
						}}
						nodeLabel={(node) => `${node.label || node.id}\n${String((node as ForceNode).data.docPath || "")}`}
						nodeColor={(node) => (node as ForceNode).color}
						nodeVal={(node) => (node as ForceNode).val || 6}
						linkColor={(link) => (link as ForceLink).color}
						linkWidth={(link) => (link as ForceLink).width}
						linkDirectionalArrowLength={(link) => ((link as ForceLink).dashed ? 0 : 4)}
						linkDirectionalArrowRelPos={1}
						onNodeClick={(node) => setSelectedNode(toGraphNode(node as ForceNode))}
						onNodeDragEnd={(node) => {
							const n = node as ForceNode;
							n.fx = n.x;
							n.fy = n.y;
							lockForceNodes(stableForceDataRef.current.nodes);
							setEngineRunning(false);
						}}
						onBackgroundClick={clearSelection}
						nodeCanvasObject={(node, ctx, globalScale) => {
							const n = node as ForceNode;
							const label = n.label || n.id;
							const fontSize = 12 / globalScale;
							const r = n.val || 6;
							const x = n.x || 0;
							const y = n.y || 0;
							const isSelected = selectedNode?.id === n.id;

							// Glow ring for selected node.
							if (isSelected) {
								ctx.beginPath();
								ctx.arc(x, y, r + 4, 0, 2 * Math.PI, false);
								ctx.strokeStyle = n.color;
								ctx.lineWidth = 2.5;
								ctx.globalAlpha = 0.6;
								ctx.stroke();
								ctx.globalAlpha = 1;
							}

							ctx.beginPath();
							ctx.arc(x, y, r, 0, 2 * Math.PI, false);
							ctx.fillStyle = n.color;
							ctx.fill();

							if (isSelected || !isLargeGraph || globalScale > 1.4) {
								ctx.font = `${fontSize}px Sans-Serif`;
								ctx.fillStyle = isDark ? "#e5e7eb" : "#111827";
								ctx.fillText(label, x + r + 2, y + fontSize / 3);
							}
						}}
					/>
				)}

				{loading && (
					<div className="absolute inset-0 flex items-center justify-center z-20 pointer-events-none bg-background/50">
						<div className="flex items-center gap-2 text-muted-foreground">
							<Loader2 className="w-5 h-5 animate-spin" />
							<span>Loading code graph...</span>
						</div>
					</div>
				)}

				{!loading && data?.nodes.length === 0 && (
					<div className="absolute inset-0 flex items-center justify-center z-10">
						<div className="text-center">
							<p className="text-sm text-muted-foreground mb-1">No code indexed yet.</p>
							<p className="text-xs text-muted-foreground">
								Run <code className="font-mono bg-muted px-1 rounded">knowns ingest</code> to index code files.
							</p>
						</div>
					</div>
				)}

				<CodeGraphLegend data={data} filters={filters} onToggleKind={toggleFilter} />

				{selectedNode && (
					<div className="absolute top-3 right-3 z-10">
						<GraphDetailPanel
							node={selectedNode}
							onClose={clearSelection}
							onNavigate={() => {}}
							onSelectNode={(id) => {
								const next = filteredData?.nodes.find((n) => n.id === id) || null;
								setSelectedNode(next);
							}}
							references={selectedNodeReferences}
						/>
					</div>
				)}
			</div>
		</div>
	);
}
