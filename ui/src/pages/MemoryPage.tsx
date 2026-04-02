import { useCallback, useEffect, useState } from "react";
import { memoryApi, type MemoryEntry } from "@/ui/api/client";
import { Brain, ChevronUp, ChevronDown, Trash2, Plus, Loader2, X } from "lucide-react";
import { cn } from "@/ui/lib/utils";

const layerColors: Record<string, { bg: string; text: string; border: string }> = {
	working: {
		bg: "bg-gray-500/10",
		text: "text-gray-600 dark:text-gray-400",
		border: "border-gray-500/30",
	},
	project: {
		bg: "bg-green-500/10",
		text: "text-green-600 dark:text-green-400",
		border: "border-green-500/30",
	},
	global: {
		bg: "bg-purple-500/10",
		text: "text-purple-600 dark:text-purple-400",
		border: "border-purple-500/30",
	},
};

const layerOrder = ["working", "project", "global"] as const;

function LayerBadge({ layer }: { layer: string }) {
	const colors = layerColors[layer] || layerColors.project;
	return (
		<span
			className={cn(
				"inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium border",
				colors.bg,
				colors.text,
				colors.border,
			)}
		>
			{layer}
		</span>
	);
}

function CategoryBadge({ category }: { category?: string }) {
	if (!category) return null;
	return (
		<span className="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium bg-muted text-muted-foreground">
			{category}
		</span>
	);
}

export default function MemoryPage() {
	const [entries, setEntries] = useState<MemoryEntry[]>([]);
	const [loading, setLoading] = useState(true);
	const [activeLayer, setActiveLayer] = useState<string>("all");
	const [selected, setSelected] = useState<MemoryEntry | null>(null);
	const [creating, setCreating] = useState(false);

	const fetchEntries = useCallback(async () => {
		try {
			const layer = activeLayer === "all" ? undefined : activeLayer;
			const data = await memoryApi.list(layer);
			setEntries(data);
		} catch (err) {
			console.error("Failed to load memories:", err);
		} finally {
			setLoading(false);
		}
	}, [activeLayer]);

	useEffect(() => {
		setLoading(true);
		fetchEntries();
	}, [fetchEntries]);

	const handlePromote = async (id: string) => {
		try {
			const updated = await memoryApi.promote(id);
			setEntries((prev) => prev.map((e) => (e.id === id ? updated : e)));
			if (selected?.id === id) setSelected(updated);
		} catch (err) {
			console.error("Failed to promote:", err);
		}
	};

	const handleDemote = async (id: string) => {
		try {
			const updated = await memoryApi.demote(id);
			setEntries((prev) => prev.map((e) => (e.id === id ? updated : e)));
			if (selected?.id === id) setSelected(updated);
		} catch (err) {
			console.error("Failed to demote:", err);
		}
	};

	const handleDelete = async (id: string) => {
		if (!confirm("Delete this memory entry?")) return;
		try {
			await memoryApi.delete(id);
			setEntries((prev) => prev.filter((e) => e.id !== id));
			if (selected?.id === id) setSelected(null);
		} catch (err) {
			console.error("Failed to delete:", err);
		}
	};

	const handleCreate = async (data: { title: string; content: string; layer: string; category: string }) => {
		try {
			const entry = await memoryApi.create(data);
			setEntries((prev) => [entry, ...prev]);
			setCreating(false);
		} catch (err) {
			console.error("Failed to create:", err);
		}
	};

	if (loading) {
		return (
			<div className="flex-1 flex items-center justify-center">
				<div className="flex items-center gap-2 text-muted-foreground">
					<Loader2 className="w-5 h-5 animate-spin" />
					<span>Loading memories...</span>
				</div>
			</div>
		);
	}

	return (
		<div className="h-full flex flex-col overflow-hidden">
			{/* Header */}
			<div className="shrink-0 px-6 pt-8 pb-4">
				<div className="flex items-center justify-between gap-4">
					<div className="flex items-baseline gap-3">
						<h1 className="text-3xl font-semibold tracking-tight">Memory</h1>
						<span className="text-sm text-muted-foreground">
							{entries.length} {entries.length === 1 ? "entry" : "entries"}
						</span>
					</div>
					<div className="flex items-center gap-2">
						{/* Layer filter */}
						<div className="flex items-center gap-0.5 rounded-lg bg-muted p-0.5">
							{["all", ...layerOrder].map((l) => (
								<button
									key={l}
									type="button"
									onClick={() => setActiveLayer(l)}
									className={cn(
										"px-2.5 py-1 rounded-md text-xs font-medium transition-colors",
										activeLayer === l
											? "bg-background text-foreground shadow-sm"
											: "text-muted-foreground hover:text-foreground",
									)}
								>
									{l.charAt(0).toUpperCase() + l.slice(1)}
								</button>
							))}
						</div>
						<button
							type="button"
							onClick={() => setCreating(true)}
							className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
						>
							<Plus className="w-3.5 h-3.5" />
							New
						</button>
					</div>
				</div>
			</div>

			{/* Content */}
			<div className="flex-1 overflow-hidden flex">
				{/* List */}
				<div className={cn("overflow-y-auto px-6 pb-6", selected ? "w-1/2" : "w-full")}>
					{creating && (
						<CreateMemoryForm
							onSubmit={handleCreate}
							onCancel={() => setCreating(false)}
						/>
					)}

					{entries.length === 0 && !creating ? (
						<div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
							<Brain className="w-10 h-10 mb-3 opacity-40" />
							<p className="text-sm">No memory entries yet.</p>
							<p className="text-xs mt-1">Create one or let AI agents store knowledge here.</p>
						</div>
					) : (
						<div className="space-y-2">
							{entries.map((entry) => (
								<div
									key={entry.id}
									onClick={() => setSelected(entry)}
									className={cn(
										"group rounded-lg border p-3 cursor-pointer transition-colors hover:bg-accent/50",
										selected?.id === entry.id
											? "border-primary/50 bg-accent/30"
											: "border-border",
									)}
								>
									<div className="flex items-start justify-between gap-2">
										<div className="min-w-0 flex-1">
											<div className="flex items-center gap-2 mb-1">
												<span className="font-medium text-sm truncate">
													{entry.title || entry.id}
												</span>
												<LayerBadge layer={entry.layer} />
												<CategoryBadge category={entry.category} />
											</div>
											{entry.content && (
												<p className="text-xs text-muted-foreground line-clamp-2">
													{entry.content}
												</p>
											)}
										</div>
										<div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
											{entry.layer !== "global" && (
												<button
													type="button"
													onClick={(e) => { e.stopPropagation(); handlePromote(entry.id); }}
													className="rounded p-1 hover:bg-accent"
													title="Promote"
												>
													<ChevronUp className="w-3.5 h-3.5" />
												</button>
											)}
											{entry.layer !== "working" && (
												<button
													type="button"
													onClick={(e) => { e.stopPropagation(); handleDemote(entry.id); }}
													className="rounded p-1 hover:bg-accent"
													title="Demote"
												>
													<ChevronDown className="w-3.5 h-3.5" />
												</button>
											)}
											<button
												type="button"
												onClick={(e) => { e.stopPropagation(); handleDelete(entry.id); }}
												className="rounded p-1 hover:bg-destructive/10 text-destructive"
												title="Delete"
											>
												<Trash2 className="w-3.5 h-3.5" />
											</button>
										</div>
									</div>
									{entry.tags && entry.tags.length > 0 && (
										<div className="flex gap-1 mt-1.5">
											{entry.tags.map((tag) => (
												<span
													key={tag}
													className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground"
												>
													{tag}
												</span>
											))}
										</div>
									)}
								</div>
							))}
						</div>
					)}
				</div>

				{/* Detail panel */}
				{selected && (
					<div className="w-1/2 border-l border-border overflow-y-auto px-6 py-4">
						<div className="flex items-center justify-between mb-4">
							<div className="flex items-center gap-2">
								<LayerBadge layer={selected.layer} />
								<CategoryBadge category={selected.category} />
								<span className="text-xs text-muted-foreground font-mono">{selected.id}</span>
							</div>
							<button
								type="button"
								onClick={() => setSelected(null)}
								className="rounded p-1 hover:bg-accent"
							>
								<X className="w-4 h-4" />
							</button>
						</div>
						<h2 className="text-xl font-semibold mb-3">{selected.title || "Untitled"}</h2>
						{selected.content && (
							<div className="prose prose-sm dark:prose-invert max-w-none">
								<pre className="whitespace-pre-wrap text-sm font-sans">{selected.content}</pre>
							</div>
						)}
						{selected.tags && selected.tags.length > 0 && (
							<div className="flex gap-1.5 mt-4 pt-4 border-t border-border">
								{selected.tags.map((tag) => (
									<span
										key={tag}
										className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground"
									>
										{tag}
									</span>
								))}
							</div>
						)}
						<div className="mt-4 pt-4 border-t border-border text-xs text-muted-foreground space-y-1">
							<p>Created: {new Date(selected.createdAt).toLocaleString()}</p>
							<p>Updated: {new Date(selected.updatedAt).toLocaleString()}</p>
						</div>
						<div className="flex gap-2 mt-4 pt-4 border-t border-border">
							{selected.layer !== "global" && (
								<button
									type="button"
									onClick={() => handlePromote(selected.id)}
									className="flex items-center gap-1 rounded-md border px-2.5 py-1.5 text-xs hover:bg-accent transition-colors"
								>
									<ChevronUp className="w-3.5 h-3.5" />
									Promote
								</button>
							)}
							{selected.layer !== "working" && (
								<button
									type="button"
									onClick={() => handleDemote(selected.id)}
									className="flex items-center gap-1 rounded-md border px-2.5 py-1.5 text-xs hover:bg-accent transition-colors"
								>
									<ChevronDown className="w-3.5 h-3.5" />
									Demote
								</button>
							)}
							<button
								type="button"
								onClick={() => handleDelete(selected.id)}
								className="flex items-center gap-1 rounded-md border border-destructive/30 px-2.5 py-1.5 text-xs text-destructive hover:bg-destructive/10 transition-colors ml-auto"
							>
								<Trash2 className="w-3.5 h-3.5" />
								Delete
							</button>
						</div>
					</div>
				)}
			</div>
		</div>
	);
}

function CreateMemoryForm({
	onSubmit,
	onCancel,
}: {
	onSubmit: (data: { title: string; content: string; layer: string; category: string }) => void;
	onCancel: () => void;
}) {
	const [title, setTitle] = useState("");
	const [content, setContent] = useState("");
	const [layer, setLayer] = useState("project");
	const [category, setCategory] = useState("");

	return (
		<div className="rounded-lg border border-primary/30 bg-accent/20 p-4 mb-4">
			<div className="flex items-center justify-between mb-3">
				<span className="text-sm font-medium">New Memory</span>
				<button type="button" onClick={onCancel} className="rounded p-1 hover:bg-accent">
					<X className="w-4 h-4" />
				</button>
			</div>
			<div className="space-y-2">
				<input
					type="text"
					placeholder="Title"
					value={title}
					onChange={(e) => setTitle(e.target.value)}
					className="w-full rounded-md border bg-background px-3 py-1.5 text-sm outline-none focus:ring-1 focus:ring-primary"
				/>
				<textarea
					placeholder="Content (markdown)"
					value={content}
					onChange={(e) => setContent(e.target.value)}
					rows={3}
					className="w-full rounded-md border bg-background px-3 py-1.5 text-sm outline-none focus:ring-1 focus:ring-primary resize-none"
				/>
				<div className="flex gap-2">
					<select
						value={layer}
						onChange={(e) => setLayer(e.target.value)}
						className="rounded-md border bg-background px-2 py-1.5 text-xs outline-none"
					>
						<option value="working">Working</option>
						<option value="project">Project</option>
						<option value="global">Global</option>
					</select>
					<input
						type="text"
						placeholder="Category (optional)"
						value={category}
						onChange={(e) => setCategory(e.target.value)}
						className="flex-1 rounded-md border bg-background px-3 py-1.5 text-xs outline-none focus:ring-1 focus:ring-primary"
					/>
				</div>
				<div className="flex justify-end gap-2 pt-1">
					<button
						type="button"
						onClick={onCancel}
						className="rounded-md px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
					>
						Cancel
					</button>
					<button
						type="button"
						onClick={() => onSubmit({ title, content, layer, category })}
						disabled={!content.trim()}
						className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
					>
						Create
					</button>
				</div>
			</div>
		</div>
	);
}
