import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { getGraph, type GraphData, type GraphNode } from "@/ui/api/client";
import { useSSEEvent } from "@/ui/contexts/SSEContext";
import { useTheme } from "@/ui/App";
import { GraphDetailPanel } from "./GraphDetailPanel";
import cytoscape from "cytoscape";
import {
	Loader2,
	Maximize2,
	Minimize2,
	RotateCcw,
	Scan,
	Search,
	X,
	Waypoints,
} from "lucide-react";
import { cn } from "@/ui/lib/utils";

const TASK_COLOR = "#3b82f6";
const DOC_COLOR = "#60a5fa";

const statusBorderColors: Record<string, string> = {
	todo: "#6b7280",
	"in-progress": "#f59e0b",
	"in-review": "#a855f7",
	done: "#22c55e",
	blocked: "#ef4444",
	"on-hold": "#8b5cf6",
	urgent: "#dc2626",
};

const memoryLayerColors: Record<string, string> = {
	working: "#6b7280",
	project: "#22c55e",
	global: "#a855f7",
};

interface FilterState {
	tasks: boolean;
	docs: boolean;
	memories: boolean;
	showIsolated: boolean;
}

function buildElements(
	data: GraphData,
	filters: FilterState,
): cytoscape.ElementDefinition[] {
	const visibleTypes = new Set<string>();
	if (filters.tasks) visibleTypes.add("task");
	if (filters.docs) visibleTypes.add("doc");
	if (filters.memories) visibleTypes.add("memory");

	const typeFilteredIds = new Set(
		data.nodes.filter((n) => visibleTypes.has(n.type)).map((n) => n.id),
	);

	const validEdges = data.edges.filter(
		(e) => typeFilteredIds.has(e.source) && typeFilteredIds.has(e.target),
	);

	const connectedIds = new Set<string>();
	for (const e of validEdges) {
		connectedIds.add(e.source);
		connectedIds.add(e.target);
	}

	const visibleNodeIds = new Set(
		[...typeFilteredIds].filter(
			(id) => filters.showIsolated || connectedIds.has(id),
		),
	);

	const nodes: cytoscape.ElementDefinition[] = data.nodes
		.filter((n) => visibleNodeIds.has(n.id))
		.map((n) => {
			const color =
				n.type === "task"
					? TASK_COLOR
					: n.type === "memory"
						? memoryLayerColors[(n.data.layer as string) || "project"] || "#22c55e"
						: DOC_COLOR;
			const borderColor =
				n.type === "task"
					? statusBorderColors[(n.data.status as string) || "todo"] || "#6b7280"
					: "transparent";
			return {
				data: {
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

function buildStylesheet(isDark: boolean): cytoscape.StylesheetStyle[] {
	const labelColor = isDark ? "#d1d5db" : "#374151";
	const dimEdge = isDark ? "#4b5563" : "#d1d5db";

	return [
		{
			selector: "node",
			style: {
				label: "data(label)",
				"text-valign": "bottom",
				"text-halign": "center",
				"font-size": "11px",
				color: labelColor,
				"text-margin-y": 4,
				"text-max-width": "100px",
				"text-wrap": "ellipsis",
				"background-color": "data(color)",
				"border-color": "data(borderColor)",
				"border-width": 2,
				"overlay-opacity": 0,
				"transition-property": "opacity, border-width, width, height",
				"transition-duration": 200,
			} as any,
		},
		{
			selector: 'node[nodeType="task"]',
			style: {
				shape: "ellipse",
				width: 28,
				height: 28,
			} as any,
		},
		{
			selector: 'node[nodeType="doc"]',
			style: {
				shape: "round-rectangle",
				width: 26,
				height: 26,
			} as any,
		},
		{
			selector: 'node[nodeType="memory"]',
			style: {
				shape: "hexagon",
				width: 24,
				height: 24,
			} as any,
		},
		{
			selector: "node:active",
			style: {
				"overlay-opacity": 0.08,
				"overlay-color": "#3b82f6",
			} as any,
		},
		{
			selector: "node.hover",
			style: {
				"border-width": 4,
				width: 34,
				height: 34,
				"z-index": 10,
			} as any,
		},
		{
			selector: "node.dimmed",
			style: {
				opacity: 0.15,
			} as any,
		},
		{
			selector: "node.highlighted",
			style: {
				opacity: 1,
				"border-width": 3,
			} as any,
		},
		{
			selector: "edge",
			style: {
				width: 1.5,
				"curve-style": "bezier",
				"target-arrow-shape": "triangle",
				"arrow-scale": 0.8,
				"line-color": dimEdge,
				"target-arrow-color": dimEdge,
				"transition-property": "opacity, line-color, width",
				"transition-duration": 200,
			} as any,
		},
		{
			selector: 'edge[edgeType="parent"]',
			style: {
				"line-color": isDark ? "#6b7280" : "#9ca3af",
				"target-arrow-color": isDark ? "#6b7280" : "#9ca3af",
				"line-style": "solid",
			} as any,
		},
		{
			selector: 'edge[edgeType="spec"]',
			style: {
				"line-color": "#3b82f6",
				"target-arrow-color": "#3b82f6",
				"line-style": "dashed",
			} as any,
		},
		{
			selector: 'edge[edgeType="mention"]',
			style: {
				"line-color": dimEdge,
				"target-arrow-color": dimEdge,
				"line-style": "dotted",
			} as any,
		},
		{
			selector: "edge.dimmed",
			style: {
				opacity: 0.1,
			} as any,
		},
		{
			selector: "edge.highlighted",
			style: {
				opacity: 1,
				width: 2.5,
			} as any,
		},
		// Search mode
		{
			selector: "node.search-match",
			style: {
				"border-width": 4,
				"border-color": "#f59e0b",
				opacity: 1,
				"z-index": 10,
			} as any,
		},
		{
			selector: "node.search-neighbor",
			style: {
				opacity: 0.8,
				"border-width": 2,
			} as any,
		},
		{
			selector: "edge.search-path",
			style: {
				opacity: 0.8,
				width: 2,
			} as any,
		},
		// Impact mode
		{
			selector: "node.impact-root",
			style: {
				"border-width": 5,
				"border-color": "#ef4444",
				opacity: 1,
				"z-index": 10,
			} as any,
		},
		{
			selector: "node.impact-hop1",
			style: {
				opacity: 1,
				"border-width": 3,
				"border-color": "#f97316",
			} as any,
		},
		{
			selector: "node.impact-hop2",
			style: {
				opacity: 0.6,
				"border-width": 2,
				"border-color": "#fbbf24",
			} as any,
		},
		{
			selector: "edge.impact-path",
			style: {
				opacity: 1,
				width: 2.5,
				"line-color": "#f97316",
				"target-arrow-color": "#f97316",
			} as any,
		},
		// Cluster mode
		{
			selector: "node.cluster-colored",
			style: {
				"background-color": "data(clusterColor)",
				"border-color": "data(clusterColor)",
				"border-width": 2,
				opacity: 1,
			} as any,
		},
	];
}

export default function GraphPage() {
	const cyRef = useRef<cytoscape.Core | null>(null);
	const containerRef = useRef<HTMLDivElement>(null);
	const { isDark } = useTheme();
	const navigate = useNavigate();

	const [data, setData] = useState<GraphData | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [filters, setFilters] = useState<FilterState>({
		tasks: true,
		docs: true,
		memories: true,
		showIsolated: false,
	});
	const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
	const [isFullscreen, setIsFullscreen] = useState(false);

	// Search state
	const [searchQuery, setSearchQuery] = useState("");
	const [searchMatchCount, setSearchMatchCount] = useState(0);
	const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

	// Impact state
	const [impactNodeId, setImpactNodeId] = useState<string | null>(null);
	const [impactSummary, setImpactSummary] = useState<{ tasks: number; docs: number } | null>(null);

	// Cluster state
	const [clustersActive, setClustersActive] = useState(false);
	const [clusterInfo, setClusterInfo] = useState<{ count: number; sizes: number[]; isolated: number } | null>(null);

	const fetchGraph = useCallback(async () => {
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

	useSSEEvent("tasks:updated", () => fetchGraph());
	useSSEEvent("tasks:refresh", () => fetchGraph());
	useSSEEvent("docs:updated", () => fetchGraph());

	const elements = useMemo(() => {
		if (!data) return [];
		return buildElements(data, filters);
	}, [data, filters]);

	// Init / update Cytoscape
	useEffect(() => {
		const container = containerRef.current;
		if (!container || loading || elements.length === 0) return;

		// Poll until container has dimensions (max 500ms)
		let attempts = 0;
		const tryInit = () => {
			if (!container.offsetWidth || !container.offsetHeight) {
				if (attempts++ < 10) {
					setTimeout(tryInit, 50);
					return;
				}
				return; // give up
			}

			if (cyRef.current) {
				cyRef.current.destroy();
			}

			const cy = cytoscape({
				container,
				elements,
				style: buildStylesheet(isDark),
				layout: {
					name: "cose",
					animate: true,
					animationDuration: 800,
					animationEasing: "ease-out-cubic",
					nodeRepulsion: () => 6000,
					idealEdgeLength: () => 60,
					gravity: 0.4,
					padding: 40,
					randomize: true,
				} as any,
				minZoom: 0.3,
				maxZoom: 5,
				wheelSensitivity: 0.3,
			});

			// Drag animation: neighbors follow with spring effect
			let draggedNode: cytoscape.NodeSingular | null = null;
			let dragStartPos: Record<string, { x: number; y: number }> = {};

			cy.on("grab", "node", (evt) => {
				draggedNode = evt.target;
				const neighbors = draggedNode!.neighborhood().nodes();
				dragStartPos = {};
				neighbors.forEach((n) => {
					const pos = n.position();
					dragStartPos[n.id()] = { x: pos.x, y: pos.y };
				});
			});

			cy.on("drag", "node", (evt) => {
				if (!draggedNode) return;
				const pos = draggedNode.position();
				const neighbors = draggedNode.neighborhood().nodes();

				neighbors.forEach((n) => {
					const startPos = dragStartPos[n.id()];
					if (!startPos || n.grabbed()) return;

					// Spring: pull neighbors 15% toward dragged node's delta
					const dx = pos.x - draggedNode!.position().x;
					const dy = pos.y - draggedNode!.position().y;
					const curPos = n.position();
					const targetX = curPos.x + dx * 0.08;
					const targetY = curPos.y + dy * 0.08;

					n.position({ x: targetX, y: targetY });
				});
			});

			cy.on("free", "node", () => {
				draggedNode = null;
				dragStartPos = {};
			});

			// Hover: highlight node + neighbors, dim rest
			cy.on("mouseover", "node", (evt) => {
				const node = evt.target;
				const neighborhood = node.closedNeighborhood();
				cy.elements().addClass("dimmed");
				neighborhood.removeClass("dimmed");
				neighborhood.nodes().addClass("highlighted");
				neighborhood.edges().addClass("highlighted");
				node.addClass("hover");
			});

			cy.on("mouseout", "node", () => {
				cy.elements().removeClass("dimmed highlighted hover");
			});

			// Click node → show detail
			cy.on("tap", "node", (evt) => {
				const nodeId = evt.target.id();
				if (data) {
					const found = data.nodes.find((n) => n.id === nodeId);
					if (found) setSelectedNode(found);
				}
			});

			// Click background → close detail
			cy.on("tap", (evt) => {
				if (evt.target === cy) setSelectedNode(null);
			});

			cyRef.current = cy;
		};

		tryInit();

		return () => {
			if (cyRef.current) {
				cyRef.current.destroy();
				cyRef.current = null;
			}
		};
	}, [elements, isDark, loading, data]);

	// Update style on theme change
	useEffect(() => {
		if (cyRef.current) {
			cyRef.current.style(buildStylesheet(isDark) as any);
		}
	}, [isDark]);

	// --- Search logic ---
	const clearGraphMode = useCallback(() => {
		const cy = cyRef.current;
		if (!cy) return;
		cy.elements().removeClass("dimmed highlighted hover search-match search-neighbor search-path impact-root impact-hop1 impact-hop2 impact-path cluster-colored");
		cy.nodes().removeData("clusterColor");
		setImpactNodeId(null);
		setImpactSummary(null);
		setClustersActive(false);
		setClusterInfo(null);
	}, []);

	const handleSearch = useCallback((query: string) => {
		setSearchQuery(query);
		if (searchTimerRef.current) clearTimeout(searchTimerRef.current);

		searchTimerRef.current = setTimeout(() => {
			const cy = cyRef.current;
			if (!cy) return;

			clearGraphMode();

			if (!query.trim()) {
				setSearchMatchCount(0);
				return;
			}

			const q = query.toLowerCase();
			const matched = cy.nodes().filter((n) => {
				const label = (n.data("label") || "").toLowerCase();
				return label.includes(q);
			});

			setSearchMatchCount(matched.length);

			if (matched.length === 0) return;

			// Get 1-hop neighbors of all matched nodes
			let neighbors = cy.collection();
			matched.forEach((n) => {
				neighbors = neighbors.union(n.closedNeighborhood());
			});

			cy.elements().addClass("dimmed");
			neighbors.removeClass("dimmed");
			matched.addClass("search-match");
			neighbors.nodes().not(matched).addClass("search-neighbor");
			neighbors.edges().addClass("search-path");
		}, 300);
	}, [clearGraphMode]);

	// --- Impact logic ---
	const handleShowImpact = useCallback((nodeId: string) => {
		const cy = cyRef.current;
		if (!cy) return;

		clearGraphMode();
		setImpactNodeId(nodeId);

		const root = cy.$id(nodeId);
		if (root.empty()) return;

		// BFS to find nodes within 2 hops
		const hop1 = root.neighborhood().nodes();
		let hop2 = cy.collection();
		hop1.forEach((n) => {
			hop2 = hop2.union(n.neighborhood().nodes());
		});
		hop2 = hop2.not(root).not(hop1);

		// Edges connecting impact nodes
		const impactNodes = root.union(hop1).union(hop2);
		const impactEdges = impactNodes.edgesWith(impactNodes);

		cy.elements().addClass("dimmed");
		impactNodes.removeClass("dimmed");
		impactEdges.removeClass("dimmed");

		root.addClass("impact-root");
		hop1.addClass("impact-hop1");
		hop2.addClass("impact-hop2");
		impactEdges.addClass("impact-path");

		// Count by type
		const allAffected = hop1.union(hop2);
		const tasks = allAffected.filter('[nodeType="task"]').length;
		const docs = allAffected.filter('[nodeType="doc"]').length;
		setImpactSummary({ tasks, docs });
	}, [clearGraphMode]);

	const handleClearImpact = useCallback(() => {
		clearGraphMode();
	}, [clearGraphMode]);

	// --- Cluster logic ---
	const CLUSTER_PALETTE = ["#ef4444", "#f59e0b", "#22c55e", "#3b82f6", "#8b5cf6", "#ec4899", "#14b8a6", "#f97316"];

	const handleToggleClusters = useCallback(() => {
		const cy = cyRef.current;
		if (!cy) return;

		if (clustersActive) {
			clearGraphMode();
			return;
		}

		clearGraphMode();
		setClustersActive(true);

		const components = cy.elements().components();
		const realClusters = components.filter((c) => c.nodes().length > 1);
		const isolatedCount = components.filter((c) => c.nodes().length === 1).length;

		realClusters
			.sort((a, b) => b.nodes().length - a.nodes().length)
			.forEach((comp, i) => {
				const color = CLUSTER_PALETTE[i % CLUSTER_PALETTE.length];
				comp.nodes().forEach((n) => {
					n.data("clusterColor", color);
					n.addClass("cluster-colored");
				});
			});

		setClusterInfo({
			count: realClusters.length,
			sizes: realClusters.map((c) => c.nodes().length),
			isolated: isolatedCount,
		});
	}, [clustersActive, clearGraphMode]);

	const toggleFilter = (key: keyof FilterState) => {
		setFilters((prev) => ({ ...prev, [key]: !prev[key] }));
	};

	const handleRelayout = () => {
		const cy = cyRef.current;
		if (!cy) return;
		cy.layout({
			name: "cose",
			animate: true,
			animationDuration: 800,
			animationEasing: "ease-out-cubic",
			nodeRepulsion: () => 6000,
			idealEdgeLength: () => 60,
			gravity: 0.4,
			padding: 40,
			randomize: false,
		} as any).run();
	};

	const handleZoomToFit = () => {
		cyRef.current?.animate({ fit: { eles: cyRef.current.elements(), padding: 40 } } as any, { duration: 400 });
	};

	const toggleFullscreen = () => {
		if (!document.fullscreenElement) {
			containerRef.current?.parentElement?.requestFullscreen();
			setIsFullscreen(true);
		} else {
			document.exitFullscreen();
			setIsFullscreen(false);
		}
	};

	const handleNodeNavigate = (node: GraphNode) => {
		const [type, ...rest] = node.id.split(":");
		const entityId = rest.join(":");
		if (type === "task") navigate({ to: `/kanban/${entityId}` });
		else if (type === "doc") navigate({ to: `/docs/${entityId}` });
	};

	if (error) {
		return (
			<div className="flex-1 flex items-center justify-center">
				<div className="text-destructive">{error}</div>
			</div>
		);
	}

	const nodeCount = elements.filter((e) => !e.data?.source).length;
	const edgeCount = elements.filter((e) => !!e.data?.source).length;

	return (
		<div className="flex-1 flex flex-col min-h-0">
			{/* Toolbar */}
			<div className="flex items-center gap-2 px-4 py-2 border-b border-border/50 bg-background/95 flex-wrap">
				<div className="flex items-center gap-1.5">
					<button
						type="button"
						onClick={() => toggleFilter("tasks")}
						className={cn(
							"flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors border",
							filters.tasks
								? "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/30"
								: "text-muted-foreground border-transparent hover:border-border",
						)}
					>
						<span className="w-2 h-2 rounded-sm bg-blue-500" />
						Tasks
					</button>
					<button
						type="button"
						onClick={() => toggleFilter("docs")}
						className={cn(
							"flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors border",
							filters.docs
								? "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/30"
								: "text-muted-foreground border-transparent hover:border-border",
						)}
					>
						<span className="w-2 h-2 rotate-45 bg-blue-500" />
						Docs
					</button>
					<button
						type="button"
						onClick={() => toggleFilter("memories")}
						className={cn(
							"flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors border",
							filters.memories
								? "bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/30"
								: "text-muted-foreground border-transparent hover:border-border",
						)}
					>
						<span className="w-2 h-2 rounded-full bg-green-500" />
						Memories
					</button>
				</div>

				<div className="h-4 w-px bg-border" />

				<button
					type="button"
					onClick={() => toggleFilter("showIsolated")}
					className={cn(
						"flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors border",
						filters.showIsolated
							? "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/30"
							: "text-muted-foreground border-transparent hover:border-border",
					)}
				>
					Isolated
				</button>

				<div className="h-4 w-px bg-border" />

				{/* Search */}
				<div className="flex items-center gap-1.5">
					<div className="relative">
						<Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3 h-3 text-muted-foreground" />
						<input
							type="text"
							value={searchQuery}
							onChange={(e) => handleSearch(e.target.value)}
							placeholder="Search graph..."
							className="h-7 w-40 rounded-md border bg-background pl-7 pr-7 text-xs placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
						/>
						{searchQuery && (
							<button
								type="button"
								onClick={() => { handleSearch(""); }}
								className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
							>
								<X className="w-3 h-3" />
							</button>
						)}
					</div>
					{searchQuery && (
						<span className="text-xs text-muted-foreground">{searchMatchCount} matches</span>
					)}
				</div>

				<div className="h-4 w-px bg-border" />

				{/* Clusters toggle */}
				<button
					type="button"
					onClick={handleToggleClusters}
					className={cn(
						"flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors border",
						clustersActive
							? "bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/30"
							: "text-muted-foreground border-transparent hover:border-border",
					)}
				>
					<Waypoints className="w-3 h-3" />
					Clusters
				</button>

				{/* Impact clear */}
				{impactNodeId && (
					<button
						type="button"
						onClick={handleClearImpact}
						className="flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors border bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/30"
					>
						<X className="w-3 h-3" />
						Clear Impact
					</button>
				)}

				<div className="flex-1" />

				<span className="text-xs text-muted-foreground">
					{nodeCount} nodes, {edgeCount} edges
				</span>

				<div className="flex items-center gap-0.5">
					<button
						type="button"
						onClick={handleRelayout}
						className="rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
						title="Re-layout"
					>
						<RotateCcw className="w-4 h-4" />
					</button>
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

			{/* Graph area */}
			<div className="flex-1 min-h-0 relative">
				<div ref={containerRef} style={{ width: "100%", height: "100%" }} />

				{/* Overlays on top of Cytoscape */}
				{loading && (
					<div className="absolute inset-0 flex items-center justify-center z-20 pointer-events-none bg-background/50">
						<div className="flex items-center gap-2 text-muted-foreground">
							<Loader2 className="w-5 h-5 animate-spin" />
							<span>Loading graph...</span>
						</div>
					</div>
				)}

				<div className="absolute bottom-3 left-3 z-10 flex flex-col gap-1 rounded-lg border bg-background/90 backdrop-blur-sm p-2.5 text-xs pointer-events-none">
					<div className="flex items-center gap-1.5">
						<span className="w-3 h-3 rounded-full bg-blue-500" />
						<span className="text-muted-foreground">Task</span>
					</div>
					<div className="flex items-center gap-1.5">
						<span className="w-3 h-3 rounded-sm bg-blue-400" />
						<span className="text-muted-foreground">Doc</span>
					</div>
					<div className="flex items-center gap-1.5">
						<span className="w-3 h-3 bg-green-500 [clip-path:polygon(50%_0%,100%_25%,100%_75%,50%_100%,0%_75%,0%_25%)]" />
						<span className="text-muted-foreground">Memory</span>
					</div>
					<div className="h-px bg-border my-0.5" />
					<div className="flex items-center gap-1.5">
						<span className="w-4 border-t-2 border-gray-400" />
						<span className="text-muted-foreground">Parent</span>
					</div>
					<div className="flex items-center gap-1.5">
						<span className="w-4 border-t-2 border-dashed border-blue-500" />
						<span className="text-muted-foreground">Spec</span>
					</div>
					<div className="flex items-center gap-1.5">
						<span className="w-4 border-t-2 border-dotted border-gray-400" />
						<span className="text-muted-foreground">Mention</span>
					</div>
				</div>

				<div className="absolute top-3 right-3 z-10">
					<GraphDetailPanel
						node={selectedNode}
						onClose={() => setSelectedNode(null)}
						onNavigate={handleNodeNavigate}
						onShowImpact={handleShowImpact}
						impactActive={!!impactNodeId}
					/>
				</div>

				{/* Impact summary overlay */}
				{impactSummary && (
					<div className="absolute top-3 left-1/2 -translate-x-1/2 z-10 rounded-lg border bg-background/95 backdrop-blur-sm shadow-lg px-4 py-2 text-xs">
						<span className="font-medium text-foreground">Impact: </span>
						<span className="text-muted-foreground">
							Affects {impactSummary.tasks} task{impactSummary.tasks !== 1 ? "s" : ""}, {impactSummary.docs} doc{impactSummary.docs !== 1 ? "s" : ""}
						</span>
					</div>
				)}

				{/* Cluster info overlay */}
				{clusterInfo && (
					<div className="absolute bottom-3 right-3 z-10 rounded-lg border bg-background/95 backdrop-blur-sm shadow-lg p-3 text-xs max-w-48">
						<div className="font-medium text-foreground mb-1.5">{clusterInfo.count} clusters</div>
						<div className="space-y-0.5">
							{clusterInfo.sizes.map((size, i) => (
								<div key={i} className="flex items-center gap-1.5">
									<span
										className="w-2 h-2 rounded-full shrink-0"
										style={{ backgroundColor: CLUSTER_PALETTE[i % CLUSTER_PALETTE.length] }}
									/>
									<span className="text-muted-foreground">{size} nodes</span>
								</div>
							))}
							{clusterInfo.isolated > 0 && (
								<div className="flex items-center gap-1.5 text-muted-foreground/60">
									<span className="w-2 h-2 rounded-full shrink-0 bg-gray-500" />
									{clusterInfo.isolated} isolated
								</div>
							)}
						</div>
					</div>
				)}
			</div>
		</div>
	);
}
