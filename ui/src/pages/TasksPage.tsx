import { useEffect, useState } from "react";
import { LayoutList, LayoutGrid } from "lucide-react";
import type { Task } from "@/ui/models/task";
import { navigateTo } from "../lib/navigation";
import { TaskDataTable } from "../components/organisms";
import { TaskDetailSheet } from "../components/organisms/TaskDetail/TaskDetailSheet";
import { ScrollArea } from "../components/ui/ScrollArea";
import { TaskGroupedView } from "./TasksPage/TaskGroupedView";

interface TasksPageProps {
	tasks: Task[];
	loading: boolean;
	onTasksUpdate: () => void;
	selectedTask?: Task | null;
	onTaskClose?: () => void;
	onNewTask: () => void;
}

type ViewMode = "table" | "grouped";

export default function TasksPage({
	tasks,
	loading,
	onTasksUpdate,
	selectedTask: externalSelectedTask,
	onTaskClose,
	onNewTask,
}: TasksPageProps) {
	const [viewMode, setViewMode] = useState<ViewMode>("table");
	const [selectedTask, setSelectedTask] = useState<Task | null>(null);

	// Handle external selected task from search
	useEffect(() => {
		if (externalSelectedTask) {
			setSelectedTask(externalSelectedTask);
		}
	}, [externalSelectedTask]);

	const handleTaskClick = (task: Task) => {
		navigateTo(`/tasks/${task.id}`);
	};

	const handleNavigateToTask = (taskId: string) => {
		navigateTo(`/tasks/${taskId}`);
	};

	if (loading) {
		return (
			<div className="flex items-center justify-center h-64">
				<div className="text-muted-foreground">Loading tasks...</div>
			</div>
		);
	}

	return (
		<div className="h-full flex flex-col overflow-hidden">
			{/* Header */}
			<div className="shrink-0 px-6 pt-8 pb-4">
				<div className="flex items-center justify-between gap-4">
					<div className="flex items-baseline gap-3 min-w-0">
						<h1 className="text-3xl font-semibold tracking-tight">Tasks</h1>
						<span className="text-sm text-muted-foreground">
							{tasks.length} {tasks.length === 1 ? "task" : "tasks"}
						</span>
					</div>

					{/* View Toggle */}
					<div className="flex items-center gap-0.5 shrink-0">
						<button
							type="button"
							onClick={() => setViewMode("table")}
							className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-sm transition-colors ${
								viewMode === "table"
									? "bg-muted text-foreground"
									: "text-muted-foreground hover:text-foreground"
							}`}
						>
							<LayoutList className="h-4 w-4" />
							<span className="hidden sm:inline">Table</span>
						</button>
						<button
							type="button"
							onClick={() => setViewMode("grouped")}
							className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-sm transition-colors ${
								viewMode === "grouped"
									? "bg-muted text-foreground"
									: "text-muted-foreground hover:text-foreground"
							}`}
						>
							<LayoutGrid className="h-4 w-4" />
							<span className="hidden sm:inline">Grouped</span>
						</button>
					</div>
				</div>
			</div>

			{/* Content */}
			<div className="flex-1 overflow-hidden px-6 pb-6">
				{viewMode === "table" ? (
					<ScrollArea className="h-full">
						<div className="pr-4">
							<TaskDataTable
								tasks={tasks}
								onTaskClick={handleTaskClick}
								onNewTask={onNewTask}
							/>
						</div>
					</ScrollArea>
				) : (
					<TaskGroupedView
						tasks={tasks}
						onTaskClick={handleTaskClick}
						onNewTask={onNewTask}
					/>
				)}
			</div>

			{/* Task Detail Sheet */}
			<TaskDetailSheet
				task={selectedTask}
				allTasks={tasks}
				onClose={() => {
					setSelectedTask(null);
					if (onTaskClose) onTaskClose();
				}}
				onUpdate={onTasksUpdate}
				onNavigateToTask={handleNavigateToTask}
			/>
		</div>
	);
}
