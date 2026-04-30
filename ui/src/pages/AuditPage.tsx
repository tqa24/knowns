import { useCallback, useEffect, useState } from "react";
import { auditApi, type AuditEvent, type AuditStats } from "@/ui/api/client";
import {
	Activity,
	AlertCircle,
	CheckCircle2,
	ChevronDown,
	ChevronRight,
	Clock,
	Filter,
	FolderOpen,
	Loader2,
	RefreshCw,
	ShieldAlert,
	BarChart3,
} from "lucide-react";
import { cn } from "@/ui/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/ui/components/ui/card";
import { ScrollArea } from "@/ui/components/ui/ScrollArea";

type Tab = "recent" | "stats";

const resultColors: Record<string, { bg: string; text: string; icon: typeof CheckCircle2 }> = {
	success: {
		bg: "bg-green-500/10",
		text: "text-green-600 dark:text-green-400",
		icon: CheckCircle2,
	},
	error: {
		bg: "bg-red-500/10",
		text: "text-red-600 dark:text-red-400",
		icon: AlertCircle,
	},
	denied: {
		bg: "bg-yellow-500/10",
		text: "text-yellow-600 dark:text-yellow-400",
		icon: ShieldAlert,
	},
};

const classColors: Record<string, string> = {
	read: "text-blue-600 dark:text-blue-400",
	write: "text-orange-600 dark:text-orange-400",
	delete: "text-red-600 dark:text-red-400",
	generate: "text-purple-600 dark:text-purple-400",
	admin: "text-gray-600 dark:text-gray-400",
};

export default function AuditPage() {
	const [tab, setTab] = useState<Tab>("recent");
	const [events, setEvents] = useState<AuditEvent[]>([]);
	const [stats, setStats] = useState<AuditStats | null>(null);
	const [loading, setLoading] = useState(true);
	const [toolFilter, setToolFilter] = useState("");
	const [resultFilter, setResultFilter] = useState("");

	const fetchRecent = useCallback(async () => {
		setLoading(true);
		try {
			const opts: Record<string, string | number> = { limit: 100 };
			if (toolFilter) opts.tool = toolFilter;
			if (resultFilter) opts.result = resultFilter;
			const data = await auditApi.recent(opts as any);
			setEvents(data.events || []);
		} catch {
			setEvents([]);
		} finally {
			setLoading(false);
		}
	}, [toolFilter, resultFilter]);

	const fetchStats = useCallback(async () => {
		setLoading(true);
		try {
			const data = await auditApi.stats();
			setStats(data);
		} catch {
			setStats(null);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		if (tab === "recent") fetchRecent();
		else fetchStats();
	}, [tab, fetchRecent, fetchStats]);

	return (
		<div className="flex-1 flex flex-col min-h-0 p-6 gap-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Activity className="w-5 h-5 text-muted-foreground" />
					<h1 className="text-xl font-semibold">MCP Audit Trail</h1>
				</div>
				<button
					onClick={() => (tab === "recent" ? fetchRecent() : fetchStats())}
					className="p-2 rounded-md hover:bg-muted transition-colors"
					title="Refresh"
				>
					<RefreshCw className={cn("w-4 h-4", loading && "animate-spin")} />
				</button>
			</div>

			{/* Tabs */}
			<div className="flex gap-1 border-b">
				<button
					onClick={() => setTab("recent")}
					className={cn(
						"px-4 py-2 text-sm font-medium border-b-2 transition-colors",
						tab === "recent"
							? "border-primary text-primary"
							: "border-transparent text-muted-foreground hover:text-foreground",
					)}
				>
					<div className="flex items-center gap-1.5">
						<Clock className="w-4 h-4" />
						Recent Activity
					</div>
				</button>
				<button
					onClick={() => setTab("stats")}
					className={cn(
						"px-4 py-2 text-sm font-medium border-b-2 transition-colors",
						tab === "stats"
							? "border-primary text-primary"
							: "border-transparent text-muted-foreground hover:text-foreground",
					)}
				>
					<div className="flex items-center gap-1.5">
						<BarChart3 className="w-4 h-4" />
						Statistics
					</div>
				</button>
			</div>

			{/* Content */}
			{tab === "recent" ? (
				<RecentTab
					events={events}
					loading={loading}
					toolFilter={toolFilter}
					resultFilter={resultFilter}
					onToolFilter={setToolFilter}
					onResultFilter={setResultFilter}
				/>
			) : (
				<StatsTab stats={stats} loading={loading} />
			)}
		</div>
	);
}

function RecentTab({
	events,
	loading,
	toolFilter,
	resultFilter,
	onToolFilter,
	onResultFilter,
}: {
	events: AuditEvent[];
	loading: boolean;
	toolFilter: string;
	resultFilter: string;
	onToolFilter: (v: string) => void;
	onResultFilter: (v: string) => void;
}) {
	// Extract unique tool names for filter.
	const tools = [...new Set(events.map((e) => e.toolName))].sort();

	return (
		<div className="flex-1 flex flex-col min-h-0 gap-3">
			{/* Filters */}
			<div className="flex items-center gap-2 text-sm">
				<Filter className="w-4 h-4 text-muted-foreground" />
				<select
					value={toolFilter}
					onChange={(e) => onToolFilter(e.target.value)}
					className="px-2 py-1 rounded border bg-background text-sm"
				>
					<option value="">All tools</option>
					{tools.map((t) => (
						<option key={t} value={t}>
							{t}
						</option>
					))}
				</select>
				<select
					value={resultFilter}
					onChange={(e) => onResultFilter(e.target.value)}
					className="px-2 py-1 rounded border bg-background text-sm"
				>
					<option value="">All results</option>
					<option value="success">Success</option>
					<option value="error">Error</option>
					<option value="denied">Denied</option>
				</select>
				<span className="text-muted-foreground ml-auto">{events.length} events</span>
			</div>

			{loading ? (
				<div className="flex-1 flex items-center justify-center">
					<Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
				</div>
			) : events.length === 0 ? (
				<div className="flex-1 flex items-center justify-center text-muted-foreground">
					No audit events found.
				</div>
			) : (
				<ScrollArea className="flex-1">
					<div className="space-y-1">
						{events.map((event, i) => (
							<EventRow key={i} event={event} />
						))}
					</div>
				</ScrollArea>
			)}
		</div>
	);
}

function EventRow({ event }: { event: AuditEvent }) {
	const [expanded, setExpanded] = useState(false);
	const rc = resultColors[event.result] ?? {
		bg: "bg-gray-500/10",
		text: "text-gray-600 dark:text-gray-400",
		icon: CheckCircle2,
	};
	const ResultIcon = rc.icon;
	const ts = new Date(event.timestamp);
	const timeStr = ts.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
	const dateStr = ts.toLocaleDateString([], { month: "short", day: "numeric" });

	const toolDisplay = event.action ? `${event.toolName}.${event.action}` : event.toolName;

	const hasDetails =
		(event.argumentSummary && Object.keys(event.argumentSummary).length > 0) ||
		event.projectRoot;

	return (
		<div
			className={cn(
				"rounded-md hover:bg-muted/50 transition-colors group",
				expanded && "bg-muted/30",
			)}
		>
			<div
				className="flex items-start gap-3 px-3 py-2 cursor-pointer"
				onClick={() => hasDetails && setExpanded(!expanded)}
			>
				{/* Expand indicator */}
				<div className="mt-1 w-3.5 flex-shrink-0">
					{hasDetails ? (
						expanded ? (
							<ChevronDown className="w-3.5 h-3.5 text-muted-foreground" />
						) : (
							<ChevronRight className="w-3.5 h-3.5 text-muted-foreground" />
						)
					) : null}
				</div>

				<div className={cn("mt-0.5 p-1 rounded", rc.bg)}>
					<ResultIcon className={cn("w-3.5 h-3.5", rc.text)} />
				</div>
				<div className="flex-1 min-w-0">
					<div className="flex items-center gap-2">
						<span className="font-mono text-sm font-medium">{toolDisplay}</span>
						<span className={cn("text-xs", classColors[event.actionClass] || "text-muted-foreground")}>
							{event.actionClass}
						</span>
						{event.dryRun && (
							<span className="text-xs px-1.5 py-0.5 rounded bg-yellow-500/10 text-yellow-600 dark:text-yellow-400">
								dry-run
							</span>
						)}
						<span className="text-xs text-muted-foreground ml-auto">
							{event.durationMs}ms
						</span>
					</div>
					{event.errorMessage && (
						<p className="text-xs text-red-500 mt-0.5 truncate">{event.errorMessage}</p>
					)}
					{event.entityRefs && event.entityRefs.length > 0 && (
						<p className="text-xs text-muted-foreground mt-0.5 truncate">
							{event.entityRefs.join(", ")}
						</p>
					)}
				</div>
				<div className="text-xs text-muted-foreground text-right whitespace-nowrap">
					<div>{timeStr}</div>
					<div>{dateStr}</div>
				</div>
			</div>

			{/* Expanded details */}
			{expanded && hasDetails && (
				<div className="px-3 pb-3 ml-10 space-y-2 border-t border-border/50 pt-2">
					{event.projectRoot && (
						<div className="flex items-center gap-1.5 text-xs text-muted-foreground">
							<FolderOpen className="w-3 h-3 flex-shrink-0" />
							<span className="font-medium">Project:</span>
							<span className="font-mono truncate">{event.projectRoot}</span>
						</div>
					)}
					{event.argumentSummary && Object.keys(event.argumentSummary).length > 0 && (
						<div>
							<p className="text-xs font-medium text-muted-foreground mb-1">Arguments</p>
							<div className="rounded-md bg-muted/50 border border-border/50 overflow-hidden">
								<table className="w-full text-xs">
									<tbody>
										{Object.entries(event.argumentSummary).map(([key, value]) => (
											<tr key={key} className="border-b border-border/30 last:border-b-0">
												<td className="px-2 py-1 font-mono font-medium text-muted-foreground whitespace-nowrap align-top">
													{key}
												</td>
												<td className="px-2 py-1 font-mono break-all">
													{value}
												</td>
											</tr>
										))}
									</tbody>
								</table>
							</div>
						</div>
					)}
				</div>
			)}
		</div>
	);
}

function StatsTab({ stats, loading }: { stats: AuditStats | null; loading: boolean }) {
	if (loading) {
		return (
			<div className="flex-1 flex items-center justify-center">
				<Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
			</div>
		);
	}

	if (!stats || stats.totalCalls === 0) {
		return (
			<div className="flex-1 flex items-center justify-center text-muted-foreground">
				No audit data available.
			</div>
		);
	}

	const toolEntries = Object.entries(stats.byTool).sort((a, b) => b[1] - a[1]);
	const classEntries = Object.entries(stats.byActionClass).sort((a, b) => b[1] - a[1]);

	return (
		<ScrollArea className="flex-1">
			<div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium text-muted-foreground">Total Calls</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="text-2xl font-bold">{stats.totalCalls}</div>
					</CardContent>
				</Card>
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium text-muted-foreground">Success Rate</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="text-2xl font-bold text-green-600 dark:text-green-400">
							{stats.totalCalls > 0
								? Math.round(((stats.byResult.success || 0) / stats.totalCalls) * 100)
								: 0}
							%
						</div>
					</CardContent>
				</Card>
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm font-medium text-muted-foreground">Dry-Run</CardTitle>
					</CardHeader>
					<CardContent>
						<div className="text-2xl font-bold">{stats.dryRunCount}</div>
						<p className="text-xs text-muted-foreground">
							vs {stats.executeCount} executed
						</p>
					</CardContent>
				</Card>
			</div>

			{/* By Result */}
			<div className="mb-6">
				<h3 className="text-sm font-semibold mb-2">By Result</h3>
				<div className="flex gap-3">
					{Object.entries(stats.byResult).map(([result, count]) => {
						const rc = resultColors[result] ?? {
							bg: "bg-gray-500/10",
							text: "text-gray-600 dark:text-gray-400",
							icon: CheckCircle2,
						};
						return (
							<div key={result} className={cn("px-3 py-2 rounded-md", rc.bg)}>
								<span className={cn("text-sm font-medium", rc.text)}>
									{result}: {count}
								</span>
							</div>
						);
					})}
				</div>
			</div>

			{/* By Action Class */}
			<div className="mb-6">
				<h3 className="text-sm font-semibold mb-2">By Action Class</h3>
				<div className="flex flex-wrap gap-2">
					{classEntries.map(([cls, count]) => (
						<div key={cls} className="px-3 py-1.5 rounded-md bg-muted">
							<span className={cn("text-sm", classColors[cls] || "text-foreground")}>
								{cls}
							</span>
							<span className="text-sm text-muted-foreground ml-1.5">{count}</span>
						</div>
					))}
				</div>
			</div>

			{/* By Tool */}
			<div>
				<h3 className="text-sm font-semibold mb-2">By Tool</h3>
				<div className="space-y-1">
					{toolEntries.map(([tool, count]) => {
						const maxCount = toolEntries[0]?.[1] || 1;
						const pct = Math.round((count / maxCount) * 100);
						const results = stats.byToolResult[tool] || {};
						return (
							<div key={tool} className="flex items-center gap-3 py-1">
								<span className="font-mono text-sm w-48 truncate">{tool}</span>
								<div className="flex-1 h-5 bg-muted rounded-full overflow-hidden">
									<div
										className="h-full bg-primary/30 rounded-full transition-all"
										style={{ width: `${pct}%` }}
									/>
								</div>
								<span className="text-sm font-medium w-12 text-right">{count}</span>
								<div className="flex gap-1 text-xs text-muted-foreground w-32">
									{Object.entries(results).map(([r, c]) => (
										<span key={r}>
											{r}:{c}
										</span>
									))}
								</div>
							</div>
						);
					})}
				</div>
			</div>
		</ScrollArea>
	);
}
