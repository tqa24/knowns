/**
 * Dashboard Page
 * Overview with charts and grid layout
 */

import { useEffect, useState, useMemo } from "react";
import {
	CheckCircle2,
	RefreshCw,
	Zap,
	Activity,
	ListTodo,
	Clock,
	TrendingUp,
	Users,
} from "lucide-react";
import type { Task } from "@/ui/models/task";
import { api, type Activity as ActivityType } from "../api/client";
import { Progress } from "../components/ui/progress";
import { cn } from "../lib/utils";

interface DashboardPageProps {
	tasks: Task[];
	loading: boolean;
}

function formatDuration(seconds: number): string {
	if (seconds < 60) return `${seconds}s`;
	const hours = Math.floor(seconds / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	if (hours > 0) return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
	return `${minutes}m`;
}

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

function getChangeDescription(change: { field: string; oldValue?: unknown; newValue?: unknown }): string {
	const { field, newValue } = change;
	switch (field) {
		case "status": return `status → ${newValue}`;
		case "priority": return `priority → ${newValue}`;
		case "assignee": return newValue ? `assigned to ${newValue}` : "unassigned";
		case "title": return "title updated";
		case "description": return "description updated";
		case "acceptanceCriteria": return "AC updated";
		default: return `${field} changed`;
	}
}

// --- Chart Components ---

interface DonutSegment {
	value: number;
	color: string;
	label: string;
}

function DonutChart({ segments, size = 140, strokeWidth = 20 }: { segments: DonutSegment[]; size?: number; strokeWidth?: number }) {
	const radius = (size - strokeWidth) / 2;
	const circumference = 2 * Math.PI * radius;
	const total = segments.reduce((sum, s) => sum + s.value, 0);
	if (total === 0) {
		return (
			<svg width={size} height={size} className="shrink-0">
				<circle cx={size / 2} cy={size / 2} r={radius} fill="none" stroke="currentColor" strokeWidth={strokeWidth} className="text-muted/30" />
			</svg>
		);
	}

	let offset = 0;
	return (
		<svg width={size} height={size} className="shrink-0 -rotate-90">
			{segments.filter(s => s.value > 0).map((seg) => {
				const pct = seg.value / total;
				const dash = pct * circumference;
				const gap = circumference - dash;
				const el = (
					<circle
						key={seg.label}
						cx={size / 2}
						cy={size / 2}
						r={radius}
						fill="none"
						stroke={seg.color}
						strokeWidth={strokeWidth}
						strokeDasharray={`${dash} ${gap}`}
						strokeDashoffset={-offset}
						strokeLinecap="round"
						className="transition-all duration-700 ease-out"
					/>
				);
				offset += dash;
				return el;
			})}
		</svg>
	);
}

function HorizontalBar({ value, max, color, label, count }: { value: number; max: number; color: string; label: string; count: number }) {
	const pct = max > 0 ? (value / max) * 100 : 0;
	return (
		<div className="flex items-center gap-3">
			<span className="text-xs text-muted-foreground w-16 shrink-0 text-right">{label}</span>
			<div className="flex-1 h-5 bg-muted/40 rounded-full overflow-hidden">
				<div
					className="h-full rounded-full transition-all duration-700 ease-out"
					style={{ width: `${pct}%`, backgroundColor: color }}
				/>
			</div>
			<span className="text-xs font-medium w-8 shrink-0">{count}</span>
		</div>
	);
}

// --- Card wrapper ---
function DashCard({ children, className }: { children: React.ReactNode; className?: string }) {
	return (
		<div className={cn("rounded-xl border border-border/50 bg-card p-5", className)}>
			{children}
		</div>
	);
}

function CardTitle({ icon: Icon, children, action }: { icon: React.ElementType; children: React.ReactNode; action?: React.ReactNode }) {
	return (
		<div className="flex items-center justify-between mb-4">
			<div className="flex items-center gap-2">
				<Icon className="w-4 h-4 text-muted-foreground" />
				<h3 className="text-sm font-semibold">{children}</h3>
			</div>
			{action}
		</div>
	);
}

// --- Main Component ---

export default function DashboardPage({ tasks, loading }: DashboardPageProps) {
	const [activities, setActivities] = useState<ActivityType[]>([]);
	const [activitiesLoading, setActivitiesLoading] = useState(true);

	useEffect(() => {
		api.getActivities({ limit: 10 })
			.then((data) => { setActivities(data); setActivitiesLoading(false); })
			.catch(() => setActivitiesLoading(false));
	}, []);

	const timeStats = useMemo(() => {
		const now = new Date();
		const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate());
		const weekStart = new Date(todayStart);
		weekStart.setDate(weekStart.getDate() - weekStart.getDay());
		let todaySeconds = 0, weekSeconds = 0, totalSeconds = 0;
		for (const task of tasks) {
			totalSeconds += task.timeSpent || 0;
			for (const entry of task.timeEntries || []) {
				const entryDate = new Date(entry.startedAt);
				if (entryDate >= todayStart) todaySeconds += entry.duration || 0;
				if (entryDate >= weekStart) weekSeconds += entry.duration || 0;
			}
		}
		return { today: todaySeconds, week: weekSeconds, total: totalSeconds };
	}, [tasks]);

	const recentTasks = useMemo(() => {
		return [...tasks]
			.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
			.slice(0, 5);
	}, [tasks]);

	const taskStats = {
		total: tasks.length,
		todo: tasks.filter((t) => t.status === "todo").length,
		inProgress: tasks.filter((t) => t.status === "in-progress").length,
		inReview: tasks.filter((t) => t.status === "in-review").length,
		done: tasks.filter((t) => t.status === "done").length,
		blocked: tasks.filter((t) => t.status === "blocked").length,
		highPriority: tasks.filter((t) => t.priority === "high" && t.status !== "done").length,
	};

	const priorityStats = {
		high: tasks.filter((t) => t.priority === "high").length,
		medium: tasks.filter((t) => t.priority === "medium").length,
		low: tasks.filter((t) => t.priority === "low").length,
	};

	const taskCompletion = taskStats.total > 0 ? Math.round((taskStats.done / taskStats.total) * 100) : 0;

	const statusSegments: DonutSegment[] = [
		{ value: taskStats.todo, color: "#9ca3af", label: "To Do" },
		{ value: taskStats.inProgress, color: "#eab308", label: "In Progress" },
		{ value: taskStats.inReview, color: "#3b82f6", label: "In Review" },
		{ value: taskStats.done, color: "#22c55e", label: "Done" },
		{ value: taskStats.blocked, color: "#ef4444", label: "Blocked" },
	];

	return (
		<div className="h-full overflow-auto">
			<div className="max-w-[1100px] mx-auto px-6 py-10">
				{/* Header */}
				<div className="mb-8">
					<h1 className="text-3xl font-semibold tracking-tight">Dashboard</h1>
					<p className="text-muted-foreground mt-1">Overview of your project</p>
				</div>

				{/* Top Metric Cards */}
				<div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-6">
					<DashCard>
						<div className="text-3xl font-semibold tracking-tight">
							{loading ? <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" /> : taskStats.total}
						</div>
						<div className="text-xs text-muted-foreground mt-1">Total Tasks</div>
					</DashCard>
					<DashCard>
						<div className="text-3xl font-semibold tracking-tight">
							{loading ? <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" /> : `${taskCompletion}%`}
						</div>
						<div className="text-xs text-muted-foreground mt-1">Completion</div>
					</DashCard>
					<DashCard>
						<div className="text-3xl font-semibold tracking-tight">
							{loading ? <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" /> : taskStats.inProgress}
						</div>
						<div className="text-xs text-muted-foreground mt-1">In Progress</div>
					</DashCard>
					<DashCard className="flex items-center gap-4">
						<div className="relative shrink-0">
							<DonutChart
								segments={[
									{ value: taskStats.done, color: "#22c55e", label: "Done" },
									{ value: taskStats.total - taskStats.done, color: "hsl(var(--muted))", label: "Remaining" },
								]}
								size={56}
								strokeWidth={8}
							/>
							<div className="absolute inset-0 flex items-center justify-center">
								<span className="text-xs font-semibold">{taskCompletion}%</span>
							</div>
						</div>
						<div>
							<div className="text-sm font-semibold">{taskStats.done}/{taskStats.total}</div>
							<div className="text-xs text-muted-foreground mt-0.5">Tasks Done</div>
						</div>
					</DashCard>
				</div>

				{/* Charts Row: Status Donut + Priority Bars */}
				<div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
					{/* Status Distribution */}
					<DashCard>
						<CardTitle icon={TrendingUp}>Status Distribution</CardTitle>
						{loading ? (
							<div className="flex items-center justify-center py-8">
								<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
							</div>
						) : (
							<div className="flex items-center gap-6">
								<div className="relative">
									<DonutChart segments={statusSegments} />
									<div className="absolute inset-0 flex flex-col items-center justify-center">
										<span className="text-2xl font-semibold">{taskCompletion}%</span>
										<span className="text-[10px] text-muted-foreground">done</span>
									</div>
								</div>
								<div className="flex-1 space-y-2">
									{statusSegments.filter(s => s.value > 0).map((seg) => (
										<div key={seg.label} className="flex items-center gap-2 text-sm">
											<div className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: seg.color }} />
											<span className="text-muted-foreground flex-1">{seg.label}</span>
											<span className="font-medium">{seg.value}</span>
										</div>
									))}
								</div>
							</div>
						)}
					</DashCard>

					{/* Priority Breakdown */}
					<DashCard>
						<CardTitle icon={Zap}>Priority Breakdown</CardTitle>
						{loading ? (
							<div className="flex items-center justify-center py-8">
								<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
							</div>
						) : (
							<div className="space-y-3 mt-2">
								<HorizontalBar value={priorityStats.high} max={taskStats.total} color="#ef4444" label="High" count={priorityStats.high} />
								<HorizontalBar value={priorityStats.medium} max={taskStats.total} color="#eab308" label="Medium" count={priorityStats.medium} />
								<HorizontalBar value={priorityStats.low} max={taskStats.total} color="#3b82f6" label="Low" count={priorityStats.low} />
								{taskStats.highPriority > 0 && (
									<div className="flex items-center gap-2 pt-2 text-xs text-red-600 dark:text-red-400">
										<Zap className="w-3 h-3" />
										<span>{taskStats.highPriority} high priority remaining</span>
									</div>
								)}
							</div>
						)}
					</DashCard>
				</div>

				{/* Middle Row: Time Tracking + Completion Progress */}
				<div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
					{/* Time Tracking */}
					<DashCard>
						<CardTitle icon={Clock}>Time Tracking</CardTitle>
						{loading ? (
							<div className="flex items-center justify-center py-8">
								<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
							</div>
						) : (
							<div className="grid grid-cols-3 gap-4">
								<div>
									<div className="text-2xl font-semibold tracking-tight">{formatDuration(timeStats.today)}</div>
									<div className="text-xs text-muted-foreground mt-1">Today</div>
								</div>
								<div>
									<div className="text-2xl font-semibold tracking-tight">{formatDuration(timeStats.week)}</div>
									<div className="text-xs text-muted-foreground mt-1">This Week</div>
								</div>
								<div>
									<div className="text-2xl font-semibold tracking-tight">{formatDuration(timeStats.total)}</div>
									<div className="text-xs text-muted-foreground mt-1">Total</div>
								</div>
							</div>
						)}
					</DashCard>

					{/* Task Completion */}
					<DashCard>
						<CardTitle icon={CheckCircle2}>Task Completion</CardTitle>
						{loading ? (
							<div className="flex items-center justify-center py-8">
								<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
							</div>
						) : (
							<div>
								<div className="flex items-center justify-between text-sm mb-2">
									<span className="text-muted-foreground">Progress</span>
									<span className="font-medium">{taskStats.done}/{taskStats.total}</span>
								</div>
								<Progress value={taskCompletion} className="h-3 mb-3" />
								<div className="flex flex-wrap gap-x-5 gap-y-1">
									{statusSegments.filter(s => s.value > 0).map((seg) => (
										<div key={seg.label} className="flex items-center gap-1.5 text-xs">
											<div className="w-2 h-2 rounded-full" style={{ backgroundColor: seg.color }} />
											<span className="text-muted-foreground">{seg.label}</span>
											<span className="font-medium">{seg.value}</span>
										</div>
									))}
								</div>
							</div>
						)}
					</DashCard>
				</div>

				{/* Charts Row: Weekly Activity + Workload */}
				<div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
					{/* Weekly Activity Bar Chart */}
					<DashCard>
						<CardTitle icon={Activity}>Weekly Activity</CardTitle>
						<WeeklyActivityChart tasks={tasks} />
					</DashCard>

					{/* Labels Distribution */}
					<DashCard>
						<CardTitle icon={Users}>Labels Overview</CardTitle>
						<LabelsChart tasks={tasks} />
					</DashCard>
				</div>

				{/* Bottom Row: Activity + Recent Tasks */}
				<div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
					{/* Recent Activity */}
					<DashCard className="flex flex-col min-w-0 overflow-hidden">
						<CardTitle icon={Activity}>Recent Activity</CardTitle>
						<div className="flex-1 min-h-[260px] min-w-0">
						{activitiesLoading ? (
							<div className="flex items-center justify-center py-8">
								<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
							</div>
						) : activities.length === 0 ? (
							<div className="flex flex-col items-center justify-center h-full text-center">
								<Activity className="w-6 h-6 text-muted-foreground/40 mb-2" />
								<p className="text-xs text-muted-foreground">No recent activity</p>
							</div>
						) : (
							<div className="space-y-0.5 max-h-[260px] overflow-y-auto overflow-x-hidden">
								{activities.slice(0, 8).map((activity, i) => (
									<a
										key={`${activity.taskId}-${activity.version}-${i}`}
										href={`/kanban/${activity.taskId}`}
										className="flex items-center gap-3 py-1.5 px-2 -mx-2 rounded-md hover:bg-muted/50 transition-colors min-w-0"
									>
										<div className="w-1.5 h-1.5 rounded-full bg-foreground/25 shrink-0" />
										<div className="flex-1 min-w-0">
											<span className="text-sm truncate block">{activity.taskTitle}</span>
											<span className="text-[11px] text-muted-foreground truncate block">
												{activity.changes.slice(0, 2).map((c) => getChangeDescription(c)).join(", ")}
											</span>
										</div>
										<span className="text-[11px] text-muted-foreground shrink-0">
											{formatRelativeTime(activity.timestamp)}
										</span>
									</a>
								))}
							</div>
						)}
						</div>
					</DashCard>

					{/* Recent Tasks */}
					<DashCard className="flex flex-col min-w-0 overflow-hidden">
						<CardTitle icon={ListTodo} action={
							<a href="/tasks" className="text-xs text-muted-foreground hover:text-foreground transition-colors">View all →</a>
						}>Recent Tasks</CardTitle>
						<div className="flex-1 min-h-[260px] min-w-0">
						{loading ? (
							<div className="flex items-center justify-center py-8">
								<RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
							</div>
						) : recentTasks.length === 0 ? (
							<div className="flex flex-col items-center justify-center h-full text-center">
								<ListTodo className="w-6 h-6 text-muted-foreground/40 mb-2" />
								<p className="text-xs text-muted-foreground">No tasks yet</p>
							</div>
						) : (
							<div className="space-y-0.5 max-h-[260px] overflow-y-auto overflow-x-hidden">
								{recentTasks.map((task) => (
									<a
										key={task.id}
										href={`/kanban/${task.id}`}
										className="flex items-center gap-3 py-1.5 px-2 -mx-2 rounded-md hover:bg-muted/50 transition-colors min-w-0"
									>
										<div className={cn(
											"w-2 h-2 rounded-full shrink-0",
											task.status === "done" ? "bg-green-500" :
											task.status === "in-progress" ? "bg-yellow-500" :
											task.status === "blocked" ? "bg-red-500" :
											task.status === "in-review" ? "bg-blue-500" : "bg-gray-400"
										)} />
										<span className="text-sm truncate flex-1">{task.title}</span>
										<span className="text-[11px] text-muted-foreground shrink-0">#{task.id}</span>
										{task.priority === "high" && (
											<span className="text-[11px] text-red-600 dark:text-red-400 shrink-0">HIGH</span>
										)}
									</a>
								))}
							</div>
						)}
						</div>
					</DashCard>
				</div>

				<div className="h-10" />
			</div>
		</div>
	);
}

// --- Weekly Activity Bar Chart ---
const DAY_LABELS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

function WeeklyActivityChart({ tasks }: { tasks: Task[] }) {
	const data = useMemo(() => {
		const now = new Date();
		const days: { label: string; created: number; updated: number }[] = [];

		for (let i = 6; i >= 0; i--) {
			const d = new Date(now);
			d.setDate(d.getDate() - i);
			const dayStart = new Date(d.getFullYear(), d.getMonth(), d.getDate());
			const dayEnd = new Date(dayStart.getTime() + 86400000);

			const created = tasks.filter((t) => {
				const c = new Date(t.createdAt);
				return c >= dayStart && c < dayEnd;
			}).length;

			const updated = tasks.filter((t) => {
				const u = new Date(t.updatedAt);
				return u >= dayStart && u < dayEnd;
			}).length;

			days.push({ label: DAY_LABELS[d.getDay()]!, created, updated });
		}
		return days;
	}, [tasks]);

	const maxVal = Math.max(1, ...data.map((d) => Math.max(d.created, d.updated)));

	return (
		<div className="flex items-end gap-2 h-[140px]">
			{data.map((day) => (
				<div key={day.label} className="flex-1 flex flex-col items-center gap-1 h-full justify-end">
					<div className="flex gap-0.5 items-end flex-1 w-full justify-center">
						<div
							className="w-2.5 rounded-t bg-blue-500/80 transition-all duration-500"
							style={{ height: `${(day.created / maxVal) * 100}%`, minHeight: day.created > 0 ? 4 : 0 }}
							title={`${day.created} created`}
						/>
						<div
							className="w-2.5 rounded-t bg-emerald-500/80 transition-all duration-500"
							style={{ height: `${(day.updated / maxVal) * 100}%`, minHeight: day.updated > 0 ? 4 : 0 }}
							title={`${day.updated} updated`}
						/>
					</div>
					<span className="text-[10px] text-muted-foreground">{day.label}</span>
				</div>
			))}
		</div>
	);
}

// --- Labels Distribution Chart ---
const LABEL_COLORS = [
	"#3b82f6", "#8b5cf6", "#ec4899", "#f97316", "#14b8a6",
	"#eab308", "#ef4444", "#6366f1", "#06b6d4", "#84cc16",
];

function LabelsChart({ tasks }: { tasks: Task[] }) {
	const labelData = useMemo(() => {
		const map = new Map<string, number>();
		for (const t of tasks) {
			for (const label of t.labels || []) {
				map.set(label, (map.get(label) || 0) + 1);
			}
		}
		return [...map.entries()]
			.sort((a, b) => b[1] - a[1])
			.slice(0, 8);
	}, [tasks]);

	const maxCount = Math.max(1, ...labelData.map(([, c]) => c));

	if (labelData.length === 0) {
		return (
			<div className="flex flex-col items-center justify-center py-8 text-center">
				<Users className="w-6 h-6 text-muted-foreground/40 mb-2" />
				<p className="text-xs text-muted-foreground">No labels yet</p>
			</div>
		);
	}

	return (
		<div className="space-y-2.5">
			{labelData.map(([label, count], i) => (
				<div key={label} className="flex items-center gap-3">
					<span className="text-xs text-muted-foreground w-20 shrink-0 truncate text-right" title={label}>
						{label}
					</span>
					<div className="flex-1 h-5 bg-muted/40 rounded-full overflow-hidden">
						<div
							className="h-full rounded-full transition-all duration-500"
							style={{
								width: `${(count / maxCount) * 100}%`,
								backgroundColor: LABEL_COLORS[i % LABEL_COLORS.length],
								opacity: 0.8,
							}}
						/>
					</div>
					<span className="text-xs font-medium w-6 shrink-0">{count}</span>
				</div>
			))}
		</div>
	);
}
