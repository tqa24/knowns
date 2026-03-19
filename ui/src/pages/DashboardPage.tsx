/**
 * Dashboard Page
 * Overview of tasks, docs, and SDD coverage — Notion-like flat layout
 */

import { useEffect, useState, useMemo } from "react";
import {
	CheckCircle2,
	AlertTriangle,
	ChevronDown,
	ChevronUp,
	RefreshCw,
	Zap,
	Activity,
	ListTodo,
	ClipboardCheck,
} from "lucide-react";
import type { Task } from "@/ui/models/task";
import { api, getDocs, getSDDStats, type SDDResult, type Activity as ActivityType } from "../api/client";
import { Button } from "../components/ui/button";
import { Progress } from "../components/ui/progress";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "../components/ui/collapsible";
import { cn, isSpec, parseACProgress, type Doc } from "../lib/utils";

interface DashboardPageProps {
	tasks: Task[];
	loading: boolean;
}

// Format duration in seconds to human readable
function formatDuration(seconds: number): string {
	if (seconds < 60) return `${seconds}s`;
	const hours = Math.floor(seconds / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	if (hours > 0) {
		return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
	}
	return `${minutes}m`;
}

// Format relative time
function formatRelativeTime(date: Date): string {
	const now = new Date();
	const diff = now.getTime() - date.getTime();
	const minutes = Math.floor(diff / 60000);
	const hours = Math.floor(diff / 3600000);
	const days = Math.floor(diff / 86400000);

	if (minutes < 1) return "just now";
	if (minutes < 60) return `${minutes}m ago`;
	if (hours < 24) return `${hours}h ago`;
	if (days < 7) return `${days}d ago`;
	return date.toLocaleDateString();
}

// Get change description
function getChangeDescription(change: { field: string; oldValue?: unknown; newValue?: unknown }): string {
	const { field, oldValue, newValue } = change;
	switch (field) {
		case "status":
			return `status → ${newValue}`;
		case "priority":
			return `priority → ${newValue}`;
		case "assignee":
			return newValue ? `assigned to ${newValue}` : "unassigned";
		case "title":
			return "title updated";
		case "description":
			return "description updated";
		case "acceptanceCriteria":
			return "AC updated";
		default:
			return `${field} changed`;
	}
}

export default function DashboardPage({ tasks, loading }: DashboardPageProps) {
	const [docs, setDocs] = useState<Doc[]>([]);
	const [docsLoading, setDocsLoading] = useState(true);
	const [sddData, setSDDData] = useState<SDDResult | null>(null);
	const [sddLoading, setSDDLoading] = useState(true);
	const [warningsOpen, setWarningsOpen] = useState(false);
	const [passedOpen, setPassedOpen] = useState(false);
	const [activities, setActivities] = useState<ActivityType[]>([]);
	const [activitiesLoading, setActivitiesLoading] = useState(true);

	// Load docs
	useEffect(() => {
		getDocs()
			.then((d) => {
				setDocs(d as unknown as Doc[]);
				setDocsLoading(false);
			})
			.catch(() => setDocsLoading(false));
	}, []);

	// Load SDD stats
	const loadSDD = async () => {
		try {
			setSDDLoading(true);
			const result = await getSDDStats();
			setSDDData(result);
		} catch (err) {
			console.error("Failed to load SDD stats:", err);
		} finally {
			setSDDLoading(false);
		}
	};

	useEffect(() => {
		loadSDD();
	}, []);

	// Load activities
	useEffect(() => {
		api.getActivities({ limit: 10 })
			.then((data) => {
				setActivities(data);
				setActivitiesLoading(false);
			})
			.catch(() => setActivitiesLoading(false));
	}, []);

	// Calculate time tracking stats
	const timeStats = useMemo(() => {
		const now = new Date();
		const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate());
		const weekStart = new Date(todayStart);
		weekStart.setDate(weekStart.getDate() - weekStart.getDay());

		let todaySeconds = 0;
		let weekSeconds = 0;
		let totalSeconds = 0;

		for (const task of tasks) {
			totalSeconds += task.timeSpent || 0;
			for (const entry of task.timeEntries || []) {
				const entryDate = new Date(entry.startedAt);
				if (entryDate >= todayStart) {
					todaySeconds += entry.duration || 0;
				}
				if (entryDate >= weekStart) {
					weekSeconds += entry.duration || 0;
				}
			}
		}

		return { today: todaySeconds, week: weekSeconds, total: totalSeconds };
	}, [tasks]);

	// Get recent tasks (sorted by updatedAt)
	const recentTasks = useMemo(() => {
		return [...tasks]
			.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
			.slice(0, 5);
	}, [tasks]);

	// Get spec progress data
	const specProgress = useMemo(() => {
		const specs = docs.filter((d) => isSpec(d));
		return specs.map((spec) => {
			const progress = parseACProgress(spec.content || "");
			const linkedTasks = tasks.filter((t) => {
				if (!t.spec) return false;
				const normalizedSpec = t.spec.replace(/\.md$/, "").replace(/^specs\//, "");
				const docPath = spec.path?.replace(/\.md$/, "").replace(/^specs\//, "") || "";
				return normalizedSpec === docPath;
			});
			const completedTasks = linkedTasks.filter((t) => t.status === "done").length;
			return {
				...spec,
				acProgress: progress,
				linkedTasks: linkedTasks.length,
				completedTasks,
			};
		}).slice(0, 6);
	}, [docs, tasks]);

	// Calculate task stats
	const taskStats = {
		total: tasks.length,
		todo: tasks.filter((t) => t.status === "todo").length,
		inProgress: tasks.filter((t) => t.status === "in-progress").length,
		inReview: tasks.filter((t) => t.status === "in-review").length,
		done: tasks.filter((t) => t.status === "done").length,
		blocked: tasks.filter((t) => t.status === "blocked").length,
		highPriority: tasks.filter((t) => t.priority === "high" && t.status !== "done").length,
	};

	const taskCompletion = taskStats.total > 0 ? Math.round((taskStats.done / taskStats.total) * 100) : 0;

	// Calculate doc stats
	const docStats = {
		total: docs.length,
	};

	const sddCoverage = sddData?.stats.coverage.percent ?? 0;

	return (
		<div className="h-full overflow-auto">
			<div className="max-w-[960px] mx-auto px-6 py-10">
				{/* Page Header */}
				<div className="mb-10">
					<h1 className="text-3xl font-semibold tracking-tight">Dashboard</h1>
					<p className="text-muted-foreground mt-1">Overview of your project</p>
				</div>

				{/* Key Metrics Row */}
				<div className="grid grid-cols-2 sm:grid-cols-4 gap-6 mb-2">
					<div>
						<div className="text-3xl font-semibold tracking-tight">
							{loading ? <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" /> : taskStats.total}
						</div>
						<div className="text-sm text-muted-foreground mt-1">Total Tasks</div>
					</div>
					<div>
						<div className="text-3xl font-semibold tracking-tight">
							{loading ? <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" /> : `${taskCompletion}%`}
						</div>
						<div className="text-sm text-muted-foreground mt-1">Completion</div>
					</div>
					<div>
						<div className="text-3xl font-semibold tracking-tight">
							{docsLoading ? <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" /> : docStats.total}
						</div>
						<div className="text-sm text-muted-foreground mt-1">Documents</div>
					</div>
					<div>
						<div className={cn(
							"text-3xl font-semibold tracking-tight",
							!sddLoading && sddData && sddCoverage >= 75 ? "text-green-600 dark:text-green-400" :
							!sddLoading && sddData && sddCoverage >= 50 ? "text-yellow-600 dark:text-yellow-400" : ""
						)}>
							{sddLoading ? <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" /> : `${sddCoverage}%`}
						</div>
						<div className="text-sm text-muted-foreground mt-1">SDD Coverage</div>
					</div>
				</div>

				{/* Tasks Section */}
				<section className="border-t border-border/40 pt-8 mt-8">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold">Tasks</h2>
						<a href="/tasks" className="text-xs text-muted-foreground hover:text-foreground transition-colors">View all →</a>
					</div>

					{loading ? (
						<div className="flex items-center justify-center py-8">
							<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
						</div>
					) : (
						<>
							{/* Completion Progress */}
							<div className="mb-5">
								<div className="flex items-center justify-between text-sm mb-2">
									<span className="text-muted-foreground">Completion</span>
									<span className="font-medium">{taskCompletion}%</span>
								</div>
								<Progress value={taskCompletion} className="h-2" />
							</div>

							{/* Status Breakdown - inline flow */}
							<div className="flex flex-wrap gap-x-6 gap-y-2">
								<div className="flex items-center gap-2">
									<div className="w-2 h-2 rounded-full bg-gray-400" />
									<span className="text-sm text-muted-foreground">To Do</span>
									<span className="text-sm font-medium">{taskStats.todo}</span>
								</div>
								<div className="flex items-center gap-2">
									<div className="w-2 h-2 rounded-full bg-yellow-500" />
									<span className="text-sm text-muted-foreground">In Progress</span>
									<span className="text-sm font-medium">{taskStats.inProgress}</span>
								</div>
								<div className="flex items-center gap-2">
									<div className="w-2 h-2 rounded-full bg-blue-500" />
									<span className="text-sm text-muted-foreground">In Review</span>
									<span className="text-sm font-medium">{taskStats.inReview}</span>
								</div>
								<div className="flex items-center gap-2">
									<div className="w-2 h-2 rounded-full bg-green-500" />
									<span className="text-sm text-muted-foreground">Done</span>
									<span className="text-sm font-medium">{taskStats.done}</span>
								</div>
								{taskStats.blocked > 0 && (
									<div className="flex items-center gap-2">
										<div className="w-2 h-2 rounded-full bg-red-500" />
										<span className="text-sm text-muted-foreground">Blocked</span>
										<span className="text-sm font-medium">{taskStats.blocked}</span>
									</div>
								)}
							</div>

							{/* High Priority Alert */}
							{taskStats.highPriority > 0 && (
								<div className="flex items-center gap-2 mt-4 text-sm text-red-600 dark:text-red-400">
									<Zap className="w-3.5 h-3.5" />
									<span>{taskStats.highPriority} high priority task{taskStats.highPriority > 1 ? "s" : ""} remaining</span>
								</div>
							)}
						</>
					)}
				</section>

				{/* Time Tracking Section */}
				<section className="border-t border-border/40 pt-8 mt-8">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold">Time Tracking</h2>
					</div>

					{loading ? (
						<div className="flex items-center justify-center py-8">
							<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
						</div>
					) : (
						<div className="grid grid-cols-3 gap-8">
							<div>
								<div className="text-2xl font-semibold tracking-tight">
									{formatDuration(timeStats.today)}
								</div>
								<div className="text-sm text-muted-foreground mt-1">Today</div>
							</div>
							<div>
								<div className="text-2xl font-semibold tracking-tight">
									{formatDuration(timeStats.week)}
								</div>
								<div className="text-sm text-muted-foreground mt-1">This Week</div>
							</div>
							<div>
								<div className="text-2xl font-semibold tracking-tight">
									{formatDuration(timeStats.total)}
								</div>
								<div className="text-sm text-muted-foreground mt-1">Total</div>
							</div>
						</div>
					)}
				</section>

				{/* Recent Activity Section */}
				<section className="border-t border-border/40 pt-8 mt-8">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold">Recent Activity</h2>
					</div>

					{activitiesLoading ? (
						<div className="flex items-center justify-center py-8">
							<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
						</div>
					) : activities.length === 0 ? (
						<div className="flex flex-col items-center justify-center py-12 text-center">
							<Activity className="w-8 h-8 text-muted-foreground/40 mb-2" />
							<p className="text-sm text-muted-foreground">No recent activity</p>
						</div>
					) : (
						<div className="space-y-0.5">
							{activities.slice(0, 5).map((activity, i) => (
								<a
									key={`${activity.taskId}-${activity.version}-${i}`}
									href={`/kanban/${activity.taskId}`}
									className="flex items-center gap-3 py-2 px-2 -mx-2 rounded-md hover:bg-muted/50 transition-colors"
								>
									<div className="w-1.5 h-1.5 rounded-full bg-foreground/25 shrink-0" />
									<div className="flex-1 min-w-0">
										<span className="text-sm truncate">{activity.taskTitle}</span>
										<span className="text-xs text-muted-foreground ml-2">
											{activity.changes.slice(0, 2).map((c) => getChangeDescription(c)).join(", ")}
										</span>
									</div>
									<span className="text-xs text-muted-foreground shrink-0">
										{formatRelativeTime(activity.timestamp)}
									</span>
								</a>
							))}
						</div>
					)}
				</section>

				{/* Recent Tasks Section */}
				<section className="border-t border-border/40 pt-8 mt-8">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold">Recent Tasks</h2>
						<a href="/tasks" className="text-xs text-muted-foreground hover:text-foreground transition-colors">View all →</a>
					</div>

					{loading ? (
						<div className="flex items-center justify-center py-8">
							<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
						</div>
					) : recentTasks.length === 0 ? (
						<div className="flex flex-col items-center justify-center py-12 text-center">
							<ListTodo className="w-8 h-8 text-muted-foreground/40 mb-2" />
							<p className="text-sm text-muted-foreground">No tasks yet</p>
						</div>
					) : (
						<div className="space-y-0.5">
							{recentTasks.map((task) => (
								<a
									key={task.id}
									href={`/kanban/${task.id}`}
									className="flex items-center gap-3 py-2 px-2 -mx-2 rounded-md hover:bg-muted/50 transition-colors"
								>
									<div className={cn(
										"w-2 h-2 rounded-full shrink-0",
										task.status === "done" ? "bg-green-500" :
										task.status === "in-progress" ? "bg-yellow-500" :
										task.status === "blocked" ? "bg-red-500" :
										task.status === "in-review" ? "bg-blue-500" : "bg-gray-400"
									)} />
									<div className="flex-1 min-w-0">
										<span className="text-sm truncate block">{task.title}</span>
									</div>
									<span className="text-xs text-muted-foreground shrink-0">#{task.id}</span>
									{task.priority === "high" && (
										<span className="text-xs text-red-600 dark:text-red-400 shrink-0">HIGH</span>
									)}
								</a>
							))}
						</div>
					)}
				</section>

				{/* SDD Coverage Section */}
				<section className="border-t border-border/40 pt-8 mt-8">
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-lg font-semibold">SDD Coverage</h2>
						<Button
							variant="ghost"
							size="sm"
							onClick={loadSDD}
							disabled={sddLoading}
							className="h-7 w-7 p-0"
						>
							<RefreshCw className={cn("w-3.5 h-3.5", sddLoading && "animate-spin")} />
						</Button>
					</div>

					{sddLoading && !sddData ? (
						<div className="flex items-center justify-center py-8">
							<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
						</div>
					) : !sddData || sddData.stats.specs.total === 0 ? (
						<div className="flex flex-col items-center justify-center py-12 text-center">
							<ClipboardCheck className="w-8 h-8 text-muted-foreground/40 mb-2" />
							<p className="text-sm text-muted-foreground">No specs found</p>
							<p className="text-xs text-muted-foreground mt-1">Create specs in docs/specs/ folder</p>
						</div>
					) : (
						<>
							{/* Coverage Stats */}
							<div className="grid grid-cols-3 gap-8 mb-5">
								<div>
									<div className="text-2xl font-semibold tracking-tight">{sddData.stats.specs.total}</div>
									<div className="text-sm text-muted-foreground mt-1">Specs</div>
								</div>
								<div>
									<div className="text-2xl font-semibold tracking-tight">{sddData.stats.tasks.withSpec}</div>
									<div className="text-sm text-muted-foreground mt-1">Linked Tasks</div>
								</div>
								<div>
									<div className={cn(
										"text-2xl font-semibold tracking-tight",
										sddCoverage >= 75 ? "text-green-600 dark:text-green-400" :
										sddCoverage >= 50 ? "text-yellow-600 dark:text-yellow-400" : "text-red-600 dark:text-red-400"
									)}>
										{sddCoverage}%
									</div>
									<div className="text-sm text-muted-foreground mt-1">Coverage</div>
								</div>
							</div>

							{/* Coverage Progress */}
							<div className="mb-5">
								<div className="flex items-center justify-between text-sm mb-2">
									<span className="text-muted-foreground">Task-Spec Coverage</span>
									<span className="font-medium">{sddData.stats.coverage.linked}/{sddData.stats.coverage.total}</span>
								</div>
								<Progress value={sddCoverage} className="h-2" />
							</div>

							{/* Warnings */}
							{sddData.warnings.length > 0 && (
								<Collapsible open={warningsOpen} onOpenChange={setWarningsOpen} className="mb-2">
									<CollapsibleTrigger className="flex items-center justify-between w-full py-1.5 text-sm hover:bg-muted/50 rounded px-2 -mx-2">
										<div className="flex items-center gap-2 text-yellow-600 dark:text-yellow-400">
											<AlertTriangle className="w-4 h-4" />
											<span>{sddData.warnings.length} Warning{sddData.warnings.length > 1 ? "s" : ""}</span>
										</div>
										{warningsOpen ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
									</CollapsibleTrigger>
									<CollapsibleContent>
										<div className="mt-2 space-y-1 max-h-32 overflow-y-auto text-xs">
											{sddData.warnings.slice(0, 5).map((w, i) => (
												<div key={`${w.entity}-${i}`} className="text-muted-foreground truncate py-0.5">
													<span className="font-mono text-yellow-600 dark:text-yellow-400">{w.entity}</span>: {w.message}
												</div>
											))}
											{sddData.warnings.length > 5 && (
												<div className="text-muted-foreground italic">+{sddData.warnings.length - 5} more</div>
											)}
										</div>
									</CollapsibleContent>
								</Collapsible>
							)}

							{/* Passed */}
							{sddData.passed.length > 0 && (
								<Collapsible open={passedOpen} onOpenChange={setPassedOpen}>
									<CollapsibleTrigger className="flex items-center justify-between w-full py-1.5 text-sm hover:bg-muted/50 rounded px-2 -mx-2">
										<div className="flex items-center gap-2 text-green-600 dark:text-green-400">
											<CheckCircle2 className="w-4 h-4" />
											<span>{sddData.passed.length} Passed</span>
										</div>
										{passedOpen ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
									</CollapsibleTrigger>
									<CollapsibleContent>
										<div className="mt-2 space-y-1 max-h-32 overflow-y-auto text-xs">
											{sddData.passed.map((p, i) => (
												<div key={`passed-${i}`} className="text-muted-foreground flex items-center gap-1.5 py-0.5">
													<CheckCircle2 className="w-3 h-3 text-green-600 dark:text-green-400 shrink-0" />
													<span className="truncate">{p}</span>
												</div>
											))}
										</div>
									</CollapsibleContent>
								</Collapsible>
							)}
						</>
					)}
				</section>

				{/* Spec Progress Section */}
				{specProgress.length > 0 && (
					<section className="border-t border-border/40 pt-8 mt-8">
						<div className="flex items-center justify-between mb-4">
							<h2 className="text-lg font-semibold">Spec Progress</h2>
							<a href="/docs" className="text-xs text-muted-foreground hover:text-foreground transition-colors">View all →</a>
						</div>

						{docsLoading ? (
							<div className="flex items-center justify-center py-8">
								<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
							</div>
						) : (
							<div className="space-y-0.5">
								{specProgress.map((spec) => {
									const acPercent = spec.acProgress.total > 0
										? Math.round((spec.acProgress.completed / spec.acProgress.total) * 100)
										: 0;
									const status = spec.metadata.status || "draft";
									return (
										<a
											key={spec.path}
											href={`/docs/${spec.path}`}
											className="flex items-center gap-4 py-2.5 px-2 -mx-2 rounded-md hover:bg-muted/50 transition-colors"
										>
											<div className="flex-1 min-w-0">
												<div className="flex items-center gap-2 mb-1">
													<span className="text-sm font-medium truncate">{spec.metadata.title}</span>
													<span className={cn(
														"text-[10px] px-1.5 py-0.5 rounded font-medium uppercase shrink-0",
														status === "implemented" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" :
														status === "approved" ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400" :
														"bg-muted text-muted-foreground"
													)}>
														{status}
													</span>
												</div>
												<Progress value={acPercent} className="h-1.5" />
											</div>
											<div className="text-right shrink-0">
												<div className={cn(
													"text-xs font-medium",
													acPercent >= 75 ? "text-green-600 dark:text-green-400" :
													acPercent >= 50 ? "text-yellow-600 dark:text-yellow-400" : "text-muted-foreground"
												)}>
													{spec.acProgress.completed}/{spec.acProgress.total} AC
												</div>
												<div className="text-xs text-muted-foreground">
													{spec.linkedTasks} tasks · {spec.completedTasks} done
												</div>
											</div>
										</a>
									);
								})}
							</div>
						)}
					</section>
				)}

				{/* Bottom spacing */}
				<div className="h-10" />
			</div>
		</div>
	);
}
