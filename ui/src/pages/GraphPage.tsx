import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import ForceGraph2D, { type ForceGraphMethods } from "react-force-graph-2d";
import { Loader2 } from "lucide-react";

import { useTheme } from "@/ui/App";
import { getGraph, type GraphData, type GraphEdge, type GraphNode } from "@/ui/api/client";
import { DocPreviewDialog } from "@/ui/components/organisms/DocsPreview/DocPreviewDialog";
import { TaskPreviewDialog } from "@/ui/components/organisms/TaskDetail/TaskPreviewDialog";
import { useSSEEvent } from "@/ui/contexts/SSEContext";

import { GraphDetailPanel } from "./GraphDetailPanel";
import { GraphLegend } from "./graph/GraphLegend";
import { GraphToolbar } from "./graph/GraphToolbar";
import { useContainerSize } from "./graph/useContainerSize";
import {
	buildSelectedNodeReferences,
	KNOWLEDGE_FILTERS,
	type FilterState,
} from "./graph/constants";

type ForceNode = GraphNode & {
	color: string;
	val?: number;
	x?: number;
	y?: number;
	vx?: number;
	vy?: number;
	highlighted?: boolean;
};

type ForceLink = GraphEdge & {
	id: string;
	source: string | ForceNode;
	target: string | ForceNode;
	color: string;
	width: number;
	dashed?: boolean;
	muted?: boolean;
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

function getNodeColor(node: GraphNode): string {
	switch (node.type) {
	case "task":
		return "#6366f1";
	case "doc":
		return "#f59e0b";
	case "memory":
		return "#22c55e";
	default:
		return "#94a3b8";
	}
}

function getEdgeColor(edge: GraphEdge): string {
	switch (edge.type) {
	case "spec":
		return "#6366f1";
	case "parent":
		return "#94a3b8";
	case "mention":
		return "#64748b";
	default:
		return "#94a3b8";
	}
}

function filterGraphData(data: GraphData, filters: FilterState): GraphData {
	const visibleNodeIds = new Set(
		data.nodes
			.filter((n) => (n.type === "task" && filters.tasks) || (n.type === "doc" && filters.docs) || (n.type === "memory" && filters.memories))
			.map((n) => n.id),
	);
	const visibleEdges = filters.showEdges
		? data.edges.filter(
				(e) =>
					visibleNodeIds.has(e.source) &&
					visibleNodeIds.has(e.target) &&
					((e.type === "parent" && filters.edgeParent) || (e.type === "spec" && filters.edgeSpec) || (e.type === "mention" && filters.edgeMention)),
			)
		: [];
	const connectedIds = new Set<string>();
	for (const edge of visibleEdges) {
		connectedIds.add(edge.source);
		connectedIds.add(edge.target);
	}
	const nodes = data.nodes.filter((n) => visibleNodeIds.has(n.id) && (filters.showIsolated || connectedIds.has(n.id)));
	return { nodes, edges: visibleEdges };
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

function buildForceData(
	data: GraphData,
	searchQuery: string,
	selectedNodeId: string | null,
): {
	nodes: ForceNode[];
	links: ForceLink[];
	matches: number;
	impactSummary: { tasks: number; docs: number } | null;
} {
	const query = searchQuery.trim().toLowerCase();
	const matchedIds = new Set<string>();
	if (query) {
		for (const node of data.nodes) {
			const haystack = `${node.label} ${node.id}`.toLowerCase();
			if (haystack.includes(query)) matchedIds.add(node.id);
		}
	}

	let highlightedIds = new Set<string>(matchedIds);
	let impactSummary: { tasks: number; docs: number } | null = null;
	if (selectedNodeId) {
		const distances = computeNeighborhood(data, selectedNodeId, 3);
		highlightedIds = new Set(distances.keys());
		const hop1to3 = [...distances.entries()].filter(([, d]) => d > 0 && d <= 3).map(([id]) => id);
		const affected = data.nodes.filter((n) => hop1to3.includes(n.id));
		impactSummary = {
			tasks: affected.filter((n) => n.type === "task").length,
			docs: affected.filter((n) => n.type === "doc").length,
		};
	}

	const nodes: ForceNode[] = data.nodes.map((node) => {
		const baseColor = getNodeColor(node);
		const active = selectedNodeId ? highlightedIds.has(node.id) : matchedIds.size > 0 ? highlightedIds.has(node.id) : true;
		return {
			...node,
			color: active ? baseColor : "rgba(148,163,184,0.25)",
			val: node.type === "task" ? 7 : node.type === "doc" ? 6.5 : 6,
			highlighted: active,
		};
	});

	const links: ForceLink[] = data.edges.map((edge) => {
		const active = selectedNodeId
			? highlightedIds.has(edge.source) && highlightedIds.has(edge.target)
			: matchedIds.size > 0
				? highlightedIds.has(edge.source) || highlightedIds.has(edge.target)
				: true;
		return {
			...edge,
			id: edgeId(edge),
			color: active ? getEdgeColor(edge) : "rgba(148,163,184,0.15)",
			width: selectedNodeId ? 2 : 1,
			dashed: edge.type === "spec" || edge.type === "mention",
			muted: !active,
		};
	});

	return {
		nodes,
		links,
		matches: matchedIds.size,
		impactSummary,
	};
}

function sameNodeIds(a: ForceNode[], b: ForceNode[]): boolean {
	if (a.length !== b.length) return false;
	for (let i = 0; i < a.length; i++) if (a[i].id !== b[i].id) return false;
	return true;
}

function sameLinkIds(a: ForceLink[], b: ForceLink[]): boolean {
	if (a.length !== b.length) return false;
	for (let i = 0; i < a.length; i++) if (a[i].id !== b[i].id) return false;
	return true;
}

export default function GraphPage() {
	const graphContainerRef = useRef<HTMLDivElement>(null);
	const graphRef = useRef<ForceGraphMethods<ForceNode, ForceLink> | undefined>(undefined);
	const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const stableForceDataRef = useRef<{ nodes: ForceNode[]; links: ForceLink[]; matches: number; impactSummary: { tasks: number; docs: number } | null }>({ ...EMPTY_FORCE_DATA, impactSummary: null });
	const { isDark } = useTheme();
	const { width, height } = useContainerSize(graphContainerRef);

	const [data, setData] = useState<GraphData | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [filters, setFilters] = useState<FilterState>(KNOWLEDGE_FILTERS);
	const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
	const [previewTaskId, setPreviewTaskId] = useState<string | null>(null);
	const [previewDocPath, setPreviewDocPath] = useState<string | null>(null);
	const [isFullscreen, setIsFullscreen] = useState(false);
	const [searchQuery, setSearchQuery] = useState("");
	const [debouncedSearchQuery, setDebouncedSearchQuery] = useState("");
	const [impactNodeId, setImpactNodeId] = useState<string | null>(null);
	const [engineRunning, setEngineRunning] = useState(false);
	const hoverNodeIdRef = useRef<string | null>(null);

	const fetchGraph = useCallback(async () => {
		setLoading(true);
		try {
			const graphData = await getGraph();
			setData(graphData);
			setError(null);
		} catch (err) {
			setError("Failed to load graph data");
			console.error(err);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		fetchGraph();
	}, [fetchGraph]);

	useSSEEvent("tasks:updated", fetchGraph);
	useSSEEvent("tasks:refresh", fetchGraph);
	useSSEEvent("docs:updated", fetchGraph);

	useEffect(() => {
		if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
		searchTimerRef.current = setTimeout(() => setDebouncedSearchQuery(searchQuery), 200);
		return () => {
			if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
		};
	}, [searchQuery]);

	const filteredData = useMemo(() => (data ? filterGraphData(data, filters) : null), [data, filters]);

	const forceData = useMemo(() => {
		if (!filteredData) return { ...EMPTY_FORCE_DATA, impactSummary: null };
		const next = buildForceData(filteredData, debouncedSearchQuery, impactNodeId);
		const prev = stableForceDataRef.current;
		const structureSame = sameNodeIds(prev.nodes, next.nodes) && sameLinkIds(prev.links, next.links);
		if (structureSame) {
			const merged = {
				nodes: prev.nodes.map((node, i) => ({ ...node, color: next.nodes[i].color, val: next.nodes[i].val, highlighted: next.nodes[i].highlighted })),
				links: prev.links.map((link, i) => ({ ...link, color: next.links[i].color, width: next.links[i].width, dashed: next.links[i].dashed, muted: next.links[i].muted })),
				matches: next.matches,
				impactSummary: next.impactSummary,
			};
			stableForceDataRef.current = merged;
			return merged;
		}
		stableForceDataRef.current = next;
		return next;
	}, [filteredData, debouncedSearchQuery, impactNodeId]);

	useEffect(() => {
		if (filteredData && (forceData.nodes.length > 0 || forceData.links.length > 0)) setEngineRunning(true);
		else setEngineRunning(false);
	}, [filteredData, forceData.nodes.length, forceData.links.length]);

	const toggleFilter = useCallback((key: keyof FilterState) => {
		setFilters((prev) => ({ ...prev, [key]: !prev[key] }));
	}, []);

	const nodeCount = forceData.nodes.length;
	const edgeCount = forceData.links.length;
	const selectedNodeReferences = useMemo(() => buildSelectedNodeReferences(filteredData, selectedNode), [filteredData, selectedNode]);

	const handleZoomToFit = useCallback(() => {
		graphRef.current?.zoomToFit(400, 40);
	}, []);

	const clearSelection = useCallback(() => {
		setSelectedNode(null);
		setImpactNodeId(null);
	}, []);

	const toggleFullscreen = useCallback(() => {
		if (!document.fullscreenElement) {
			document.documentElement.requestFullscreen();
			setIsFullscreen(true);
		} else {
			document.exitFullscreen();
			setIsFullscreen(false);
		}
	}, []);

	const handleNodeNavigate = useCallback((node: GraphNode) => {
		const [type, ...rest] = node.id.split(":");
		const entityId = rest.join(":");
		if (type === "task") setPreviewTaskId(entityId);
		else if (type === "doc") setPreviewDocPath(entityId);
	}, []);

	const handleShowImpact = useCallback((nodeId: string) => {
		setImpactNodeId(nodeId);
	}, []);

	const handleClearImpact = useCallback(() => {
		setImpactNodeId(null);
	}, []);

	if (error) {
		return (
			<div className="flex-1 flex items-center justify-center">
				<div className="text-destructive">{error}</div>
			</div>
		);
	}

	return (
		<div className="flex-1 flex flex-col min-h-0 bg-background">
			<GraphToolbar
				filters={filters}
				searchQuery={searchQuery}
				searchMatchCount={forceData.matches}
				impactNodeId={impactNodeId}
				isFullscreen={isFullscreen}
				nodeCount={nodeCount}
				edgeCount={edgeCount}
				onToggleFilter={toggleFilter}
				onSearchChange={setSearchQuery}
				onClearImpact={handleClearImpact}
				onZoomToFit={handleZoomToFit}
				onToggleFullscreen={toggleFullscreen}
			/>

			<div className="flex-1 flex min-h-0">
				{/* Graph canvas */}
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
							cooldownTicks={120}
							cooldownTime={4000}
							onEngineStop={() => {
								lockForceNodes(stableForceDataRef.current.nodes);
								setEngineRunning(false);
							}}
							nodeLabel={() => ""}
							nodeColor={(node) => (node as ForceNode).color}
							nodeVal={(node) => (node as ForceNode).val || 6}
							linkColor={(link) => (link as ForceLink).color}
							linkWidth={(link) => (link as ForceLink).width}
							linkDirectionalArrowLength={3.5}
							linkDirectionalArrowRelPos={1}
							onNodeClick={(node) => {
								const gn = { id: node.id, label: node.label || String(node.id), type: (node as ForceNode).type, data: (node as ForceNode).data };
								setSelectedNode(gn);
								setImpactNodeId(node.id);
							}}
							onNodeHover={(node) => {
								hoverNodeIdRef.current = node ? node.id : null;
							}}
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

								if (isSelected || n.id === hoverNodeIdRef.current) {
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
								<span>Loading graph...</span>
							</div>
						</div>
					)}

					{engineRunning && (
						<div className="absolute left-3 top-3 z-20 rounded-md border bg-background/90 px-3 py-1.5 text-xs text-muted-foreground shadow-sm backdrop-blur-sm">
							Running layout...
						</div>
					)}

					<GraphLegend data={data} filters={filters} onToggleFilter={toggleFilter} />

					<div className="absolute top-3 right-3 z-10">
						<GraphDetailPanel
							node={selectedNode}
							onClose={clearSelection}
							onNavigate={handleNodeNavigate}
							onShowImpact={handleShowImpact}
							onSelectNode={(id) => {
								const next = filteredData?.nodes.find((n) => n.id === id) || null;
								setSelectedNode(next);
							}}
							impactActive={!!impactNodeId}
							references={selectedNodeReferences}
						/>
					</div>

					{forceData.impactSummary && (
						<div className="absolute top-3 left-1/2 -translate-x-1/2 z-10 rounded-lg border bg-background/95 backdrop-blur-sm shadow-lg px-4 py-2 text-xs">
							<span className="font-medium text-foreground">Impact: </span>
							<span className="text-muted-foreground">
								Affects {forceData.impactSummary.tasks} task{forceData.impactSummary.tasks !== 1 ? "s" : ""}, {forceData.impactSummary.docs} doc{forceData.impactSummary.docs !== 1 ? "s" : ""}
							</span>
						</div>
					)}
				</div>
			</div>

			<TaskPreviewDialog
				taskId={previewTaskId}
				open={!!previewTaskId}
				onOpenChange={(open) => {
					if (!open) setPreviewTaskId(null);
				}}
			/>
			<DocPreviewDialog
				docPath={previewDocPath}
				open={!!previewDocPath}
				onOpenChange={(open) => {
					if (!open) setPreviewDocPath(null);
				}}
			/>
		</div>
	);
}
