import { useState } from "react";
import { Plus, Archive, ChevronDown, ListTodo, X } from "lucide-react";
import type { Task } from "@/ui/models/task";
import { Board } from "../components/organisms";
import { Button } from "../components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "../components/ui/DropdownMenu";
import { api } from "../api/client";
import { toast } from "../components/ui/sonner";
import { useIsMobile } from "../hooks/useMobile";

// Time duration options for batch archive (in milliseconds)
const BATCH_ARCHIVE_OPTIONS = [
	{ label: "1 hour ago", value: 1 * 60 * 60 * 1000 },
	{ label: "1 day ago", value: 24 * 60 * 60 * 1000 },
	{ label: "1 week ago", value: 7 * 24 * 60 * 60 * 1000 },
	{ label: "1 month ago", value: 30 * 24 * 60 * 60 * 1000 },
	{ label: "3 months ago", value: 90 * 24 * 60 * 60 * 1000 },
];

interface KanbanPageProps {
	tasks: Task[];
	loading: boolean;
	onTasksUpdate: (tasks: Task[]) => void;
	onNewTask: () => void;
}

export default function KanbanPage({ tasks, loading, onTasksUpdate, onNewTask }: KanbanPageProps) {
	const isMobile = useIsMobile();
	const [mobileWarningDismissed, setMobileWarningDismissed] = useState(() => {
		return sessionStorage.getItem("kanban-mobile-warning-dismissed") === "true";
	});

	// Count done tasks for batch archive preview
	const getDoneTasksCount = (olderThanMs: number): number => {
		const cutoffTime = Date.now() - olderThanMs;
		return tasks.filter(
			(t) => t.status === "done" && new Date(t.updatedAt).getTime() < cutoffTime
		).length;
	};

	const handleBatchArchive = async (olderThanMs: number, label: string) => {
		try {
			const result = await api.batchArchiveTasks(olderThanMs);
			if (result.count > 0) {
				// Remove archived tasks from list
				const archivedIds = new Set(result.tasks.map((t) => t.id));
				onTasksUpdate(tasks.filter((t) => !archivedIds.has(t.id)));
				toast.success(`Archived ${result.count} task${result.count > 1 ? "s" : ""} done before ${label}`);
			} else {
				toast.info(`No done tasks found before ${label}`);
			}
		} catch (error) {
			console.error("Failed to batch archive tasks:", error);
			toast.error("Failed to archive tasks");
		}
	};

	const dismissMobileWarning = () => {
		setMobileWarningDismissed(true);
		sessionStorage.setItem("kanban-mobile-warning-dismissed", "true");
	};

	return (
		<div className="flex flex-col h-full overflow-hidden">
			{/* Mobile Warning Banner */}
			{isMobile && !mobileWarningDismissed && (
				<div className="shrink-0 mx-4 mt-4 py-2 px-3 flex items-center gap-3 text-sm text-yellow-700 dark:text-yellow-300 bg-yellow-50/50 dark:bg-yellow-900/10 rounded-md">
					<ListTodo className="w-4 h-4 shrink-0" />
					<span className="flex-1">
						Drag & drop may not work well on mobile.{" "}
						<a href="/tasks" className="underline font-medium">Use Tasks page</a> instead.
					</span>
					<button
						type="button"
						onClick={dismissMobileWarning}
						className="shrink-0 p-0.5 text-yellow-600 dark:text-yellow-400 hover:text-yellow-800 dark:hover:text-yellow-200"
					>
						<X className="w-3.5 h-3.5" />
					</button>
				</div>
			)}

			{/* Header */}
			<div className="shrink-0 px-6 pt-8 pb-4">
				<div className="flex items-center justify-between gap-4">
					<div>
						<h1 className="text-3xl font-semibold tracking-tight">Kanban Board</h1>
					</div>
					<div className="flex items-center gap-2 shrink-0">
						{/* Batch Archive Dropdown */}
						<DropdownMenu>
							<DropdownMenuTrigger asChild>
								<Button variant="ghost" size="sm" className="gap-1.5 text-muted-foreground hover:text-foreground">
									<Archive className="w-4 h-4" />
									<span className="hidden sm:inline text-sm">Archive</span>
									<ChevronDown className="w-3 h-3" />
								</Button>
							</DropdownMenuTrigger>
							<DropdownMenuContent align="end">
								{BATCH_ARCHIVE_OPTIONS.map((option) => {
									const count = getDoneTasksCount(option.value);
									return (
										<DropdownMenuItem
											key={option.value}
											onClick={() => handleBatchArchive(option.value, option.label)}
											disabled={count === 0}
										>
											<span className="flex-1">Done before {option.label}</span>
											<span className="ml-2 text-xs text-muted-foreground">
												({count})
											</span>
										</DropdownMenuItem>
									);
								})}
							</DropdownMenuContent>
						</DropdownMenu>
						{/* New Task Button */}
						<Button
							onClick={onNewTask}
							size="sm"
							className="gap-1.5"
						>
							<Plus className="w-4 h-4" />
							<span className="hidden sm:inline text-sm">New Task</span>
						</Button>
					</div>
				</div>
			</div>

			{/* Board with scrollable columns */}
			<div className="flex-1 overflow-hidden px-6 pb-6">
				<Board tasks={tasks} loading={loading} onTasksUpdate={onTasksUpdate} />
			</div>
		</div>
	);
}
