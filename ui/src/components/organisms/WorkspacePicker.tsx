import { useState, useEffect, useCallback } from "react";
import { FolderOpen, Trash2, Search, Plus, ArrowRightLeft, Clock } from "lucide-react";
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogDescription,
} from "@/ui/components/ui/dialog";
import { Button } from "@/ui/components/ui/button";
import { Input } from "@/ui/components/ui/input";
import { workspaceApi, type WorkspaceProject } from "@/ui/api/client";

interface WorkspacePickerProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
}

function formatRelativeTime(dateStr: string): string {
	const date = new Date(dateStr);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffMin = Math.floor(diffMs / 60000);
	if (diffMin < 1) return "just now";
	if (diffMin < 60) return `${diffMin}m ago`;
	const diffHr = Math.floor(diffMin / 60);
	if (diffHr < 24) return `${diffHr}h ago`;
	const diffDay = Math.floor(diffHr / 24);
	if (diffDay < 30) return `${diffDay}d ago`;
	return date.toLocaleDateString();
}

export function WorkspacePicker({ open, onOpenChange }: WorkspacePickerProps) {
	const [projects, setProjects] = useState<WorkspaceProject[]>([]);
	const [loading, setLoading] = useState(false);
	const [scanning, setScanning] = useState(false);
	const [switching, setSwitching] = useState<string | null>(null);
	const [addPath, setAddPath] = useState("");
	const [scanDirs, setScanDirs] = useState("");
	const [error, setError] = useState<string | null>(null);

	const loadProjects = useCallback(async () => {
		setLoading(true);
		setError(null);
		try {
			const data = await workspaceApi.list();
			setProjects(data || []);
		} catch (err) {
			setError("Failed to load projects");
			console.error(err);
		} finally {
			setLoading(false);
		}
	}, []);

	const handleAutoScan = useCallback(async () => {
		setScanning(true);
		try {
			const added = await workspaceApi.autoScan();
			if (added.length > 0) {
				await loadProjects();
			}
		} catch (err) {
			console.error("Auto-scan failed:", err);
		} finally {
			setScanning(false);
		}
	}, [loadProjects]);

	useEffect(() => {
		if (open) {
			loadProjects();
			handleAutoScan();
		}
	}, [open, loadProjects, handleAutoScan]);

	const handleSwitch = async (id: string) => {
		setSwitching(id);
		setError(null);
		try {
			await workspaceApi.switchProject(id);
			onOpenChange(false);
		} catch (err) {
			setError("Failed to switch workspace");
			console.error(err);
		} finally {
			setSwitching(null);
		}
	};

	const handleRemove = async (id: string) => {
		setError(null);
		try {
			await workspaceApi.remove(id);
			setProjects((prev) => prev.filter((p) => p.id !== id));
		} catch (err) {
			setError("Failed to remove project");
			console.error(err);
		}
	};

	const handleScan = async () => {
		if (!scanDirs.trim()) return;
		setError(null);
		try {
			const dirs = scanDirs.split(",").map((d) => d.trim()).filter(Boolean);
			const added = await workspaceApi.scan(dirs);
			if (added.length > 0) {
				await loadProjects();
			}
			setScanDirs("");
		} catch (err) {
			setError("Failed to scan directories");
			console.error(err);
		}
	};

	const handleAdd = async () => {
		if (!addPath.trim()) return;
		setError(null);
		try {
			// Scan the parent dir to discover the project
			const dirs = [addPath.trim()];
			await workspaceApi.scan(dirs);
			await loadProjects();
			setAddPath("");
		} catch (err) {
			setError("Failed to add project");
			console.error(err);
		}
	};

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-lg">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<ArrowRightLeft className="h-5 w-5" />
						Switch Workspace
					</DialogTitle>
					<DialogDescription>
						Select a project or add new ones to the registry.
					</DialogDescription>
				</DialogHeader>

				{error && (
					<div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
						{error}
					</div>
				)}

				{scanning && (
					<div className="flex items-center gap-2 rounded-md bg-primary/5 px-3 py-2 text-sm text-muted-foreground">
						<Search className="h-3.5 w-3.5 animate-pulse" />
						Scanning common directories...
					</div>
				)}

				{/* Project List */}
				<div className="max-h-64 space-y-1 overflow-y-auto">
					{loading ? (
						<div className="py-8 text-center text-sm text-muted-foreground">Loading...</div>
					) : projects.length === 0 ? (
						<div className="py-8 text-center text-sm text-muted-foreground">
							No projects registered. Scan a directory or add a path below.
						</div>
					) : (
						projects.map((project) => (
							<div
								key={project.id}
								className="group flex items-center gap-3 rounded-lg border px-3 py-2.5 hover:bg-accent/50 transition-colors cursor-pointer"
								onClick={() => handleSwitch(project.id)}
								onKeyDown={(e) => e.key === "Enter" && handleSwitch(project.id)}
								role="button"
								tabIndex={0}
							>
								<FolderOpen className="h-4 w-4 shrink-0 text-muted-foreground" />
								<div className="min-w-0 flex-1">
									<div className="truncate text-sm font-medium">{project.name}</div>
									<div className="truncate text-xs text-muted-foreground">{project.path}</div>
								</div>
								<div className="flex items-center gap-1.5 shrink-0">
									<span className="flex items-center gap-1 text-xs text-muted-foreground">
										<Clock className="h-3 w-3" />
										{formatRelativeTime(project.lastUsed)}
									</span>
									{switching === project.id && (
										<span className="text-xs text-primary">Switching...</span>
									)}
									<button
										type="button"
										className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-destructive/10 hover:text-destructive transition-all"
										onClick={(e) => {
											e.stopPropagation();
											handleRemove(project.id);
										}}
										aria-label={`Remove ${project.name}`}
									>
										<Trash2 className="h-3.5 w-3.5" />
									</button>
								</div>
							</div>
						))
					)}
				</div>

				{/* Add Project */}
				<div className="flex gap-2">
					<Input
						placeholder="Project parent directory path..."
						value={addPath}
						onChange={(e) => setAddPath(e.target.value)}
						onKeyDown={(e) => e.key === "Enter" && handleAdd()}
						className="text-sm"
					/>
					<Button variant="outline" size="sm" onClick={handleAdd} disabled={!addPath.trim()}>
						<Plus className="h-4 w-4 mr-1" />
						Add
					</Button>
				</div>

				{/* Scan */}
				<div className="flex gap-2">
					<Input
						placeholder="Scan directories (comma-separated)..."
						value={scanDirs}
						onChange={(e) => setScanDirs(e.target.value)}
						onKeyDown={(e) => e.key === "Enter" && handleScan()}
						className="text-sm"
					/>
					<Button variant="outline" size="sm" onClick={handleScan} disabled={!scanDirs.trim()}>
						<Search className="h-4 w-4 mr-1" />
						Scan
					</Button>
				</div>
			</DialogContent>
		</Dialog>
	);
}
