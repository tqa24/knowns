import { useCallback, useEffect, useState, type ReactNode } from "react";
import {
	memoryApi,
	workingMemoryApi,
	type MemoryEntry,
	type PersistentMemoryLayer,
} from "@/ui/api/client";
import {
	Brain,
	ChevronDown,
	ChevronUp,
	Loader2,
	Pencil,
	Plus,
	RefreshCw,
	Trash2,
} from "lucide-react";
import { cn } from "@/ui/lib/utils";
import MDRender from "@/ui/components/editor/MDRender";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/ui/components/ui/dialog";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/ui/components/ui/card";
import { ScrollArea } from "@/ui/components/ui/ScrollArea";

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

const persistentLayerOrder: PersistentMemoryLayer[] = ["project", "global"];

type MemoryScope = "persistent" | "working";
type ModalMode = "view" | "create" | "edit" | null;

const scopeMeta: Record<MemoryScope, { label: string; description: string; emptyTitle: string; emptyDescription: string }> = {
	persistent: {
		label: "Memories",
		description: "Project and global knowledge that persists across sessions.",
		emptyTitle: "No persistent memories yet",
		emptyDescription: "Save durable notes, decisions, and reusable context here.",
	},
	working: {
		label: "Working memory",
		description: "Temporary notes for the current session only.",
		emptyTitle: "No working memory",
		emptyDescription: "Capture short-lived thoughts without saving them permanently.",
	},
};

function formatDate(value?: string, includeTime = false) {
	if (!value) return "—";
	return new Date(value).toLocaleString(
		undefined,
		includeTime
			? { dateStyle: "medium", timeStyle: "short" }
			: { dateStyle: "medium" },
	);
}

function LayerBadge({ layer }: { layer: string }) {
	const colors = layerColors[layer] || layerColors.project;
	return (
		<span
			className={cn(
				"inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium",
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
		<span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
			{category}
		</span>
	);
}

export default function MemoryPage() {
	const [persistentEntries, setPersistentEntries] = useState<MemoryEntry[]>([]);
	const [workingEntries, setWorkingEntries] = useState<MemoryEntry[]>([]);
	const [loading, setLoading] = useState(true);
	const [scope, setScope] = useState<MemoryScope>("persistent");
	const [activeLayer, setActiveLayer] = useState<"all" | PersistentMemoryLayer>("all");
	const [selectedEntry, setSelectedEntry] = useState<MemoryEntry | null>(null);
	const [modalMode, setModalMode] = useState<ModalMode>(null);

	const fetchEntries = useCallback(async () => {
		try {
			const layer = activeLayer === "all" ? undefined : activeLayer;
			const persistent = await memoryApi.list(layer);
			setPersistentEntries(persistent);
		} catch (err) {
			console.error("Failed to load memory:", err);
		} finally {
			setLoading(false);
		}
	}, [activeLayer]);

	useEffect(() => {
		setLoading(true);
		void fetchEntries();
	}, [fetchEntries]);

	const visibleEntries = persistentEntries;
	const persistentCount = persistentEntries.length;
	const meta = scopeMeta[scope];
	const dialogOpen = modalMode !== null;
	const selectedPersistentEntry =
		scope === "persistent" && selectedEntry && selectedEntry.layer !== "working"
			? selectedEntry
			: null;

	useEffect(() => {
		if (!selectedEntry) return;
		const stillVisible = visibleEntries.some((entry) => entry.id === selectedEntry.id);
		if (!stillVisible) {
			setSelectedEntry(null);
			setModalMode(null);
		}
	}, [visibleEntries, selectedEntry]);

	const closeModal = () => {
		setModalMode(null);
		setSelectedEntry(null);
	};

	const replacePersistentEntry = (entry: MemoryEntry) => {
		setPersistentEntries((prev) => prev.map((item) => (item.id === entry.id ? entry : item)));
		if (selectedEntry?.id === entry.id) {
			setSelectedEntry(entry);
		}
	};

	const handlePromote = async (id: string) => {
		try {
			const updated = await memoryApi.promote(id);
			replacePersistentEntry(updated);
		} catch (err) {
			console.error("Failed to promote memory:", err);
		}
	};

	const handleDemote = async (id: string) => {
		try {
			const updated = await memoryApi.demote(id);
			replacePersistentEntry(updated);
		} catch (err) {
			console.error("Failed to demote memory:", err);
		}
	};

	const handleDeletePersistent = async (id: string) => {
		if (!confirm("Delete this memory entry?")) return;
		try {
			await memoryApi.delete(id);
			setPersistentEntries((prev) => prev.filter((entry) => entry.id !== id));
			if (selectedEntry?.id === id) {
				closeModal();
			}
		} catch (err) {
			console.error("Failed to delete memory:", err);
		}
	};

	const handleDeleteWorking = async (id: string) => {
		if (!confirm("Delete this working memory entry?")) return;
		try {
			await workingMemoryApi.delete(id);
			setWorkingEntries((prev) => prev.filter((entry) => entry.id !== id));
			if (selectedEntry?.id === id) {
				closeModal();
			}
		} catch (err) {
			console.error("Failed to delete working memory:", err);
		}
	};

	const handleClearWorking = async () => {
		if (!confirm("Clear all working memory entries for this session?")) return;
		try {
			await workingMemoryApi.clean();
			setWorkingEntries([]);
			closeModal();
		} catch (err) {
			console.error("Failed to clear working memory:", err);
		}
	};

	const handleCreatePersistent = async (data: {
		title: string;
		content: string;
		layer: PersistentMemoryLayer;
		category: string;
	}) => {
		try {
			const entry = await memoryApi.create(data);
			setPersistentEntries((prev) => [entry, ...prev]);
			setSelectedEntry(entry);
			setModalMode("view");
		} catch (err) {
			console.error("Failed to create memory:", err);
		}
	};

	const handleCreateWorking = async (data: { title: string; content: string; category: string }) => {
		try {
			const entry = await workingMemoryApi.create({ ...data, layer: "working" });
			setWorkingEntries((prev) => [entry, ...prev]);
			setSelectedEntry(entry);
			setModalMode("view");
		} catch (err) {
			console.error("Failed to create working memory:", err);
		}
	};

	const handleUpdatePersistent = async (id: string, data: { title: string; content: string; category: string }) => {
		try {
			const entry = await memoryApi.update(id, data);
			replacePersistentEntry(entry);
			setSelectedEntry(entry);
			setModalMode("view");
		} catch (err) {
			console.error("Failed to update memory:", err);
		}
	};

	if (loading) {
		return (
			<div className="flex flex-1 items-center justify-center">
				<div className="flex items-center gap-2 text-muted-foreground">
					<Loader2 className="h-5 w-5 animate-spin" />
					<span>Loading memory...</span>
				</div>
			</div>
		);
	}

	return (
		<>
			<div className="flex h-full flex-col overflow-hidden">
				<div className="shrink-0 px-6 pt-8 pb-4">
					<div className="flex flex-wrap items-start justify-between gap-4">
						<div className="space-y-2">
							<div className="flex flex-wrap items-baseline gap-3">
								<h1 className="text-3xl font-semibold tracking-tight">Memories</h1>
								<span className="text-sm text-muted-foreground">
									{persistentCount} total entries
								</span>
							</div>
							<p className="text-sm text-muted-foreground">
								Grid view for browsing, modal for reading and editing.
							</p>
						</div>
						<button
							type="button"
							onClick={() => {
								setLoading(true);
								void fetchEntries();
							}}
							className="inline-flex items-center gap-2 rounded-lg border border-border/60 bg-background px-3 py-2 text-sm font-medium hover:bg-accent transition-colors"
						>
							<RefreshCw className="h-4 w-4" />
							Refresh
						</button>
					</div>
				</div>

				<div className="min-h-0 flex-1 px-6 pb-6">
					<Card className="flex h-full min-h-0 flex-col overflow-hidden border-border/60 shadow-sm">
						<CardHeader className="gap-4 border-b border-border/50">
							<div className="flex flex-wrap items-start justify-between gap-4">
								<div className="grid min-w-[280px] flex-1 gap-2 lg:max-w-md">
									<ScopeButton
										label={`Memories (${persistentCount})`}
										description="Project and global"
										active={true}
										onClick={() => {
											closeModal();
										}}
									/>
								</div>

								<div className="flex flex-wrap items-center gap-2">
									<div className="flex flex-wrap gap-1.5">
										{["all", ...persistentLayerOrder].map((layer) => (
											<FilterChip
												key={layer}
												active={activeLayer === layer}
												onClick={() => setActiveLayer(layer as "all" | PersistentMemoryLayer)}
											>
												{layer.charAt(0).toUpperCase() + layer.slice(1)}
											</FilterChip>
										))}
									</div>
									<button
										type="button"
										onClick={() => {
											setSelectedEntry(null);
											setModalMode("create");
										}}
										className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
									>
										<Plus className="h-4 w-4" />
										New
									</button>
								</div>
							</div>
							<div>
								<CardTitle className="text-base">Memories</CardTitle>
								<CardDescription className="mt-1">Project and global knowledge that persists across sessions.</CardDescription>
							</div>
						</CardHeader>

						<CardContent className="min-h-0 flex-1 p-0">
							<ScrollArea className="h-full">
								<div className="p-4 sm:p-5">
									{visibleEntries.length === 0 ? (
										<EmptyGridState title={meta.emptyTitle} description={meta.emptyDescription} />
									) : (
										<div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
											{visibleEntries.map((entry) => (
												<MemoryGridCard
													key={entry.id}
													entry={entry}
													onOpen={() => {
														setSelectedEntry(entry);
														setModalMode("view");
													}}
													onDelete={() =>
														scope === "persistent"
															? handleDeletePersistent(entry.id)
															: handleDeleteWorking(entry.id)
													}
													onPromote={scope === "persistent" && entry.layer !== "global" ? () => handlePromote(entry.id) : undefined}
													onDemote={scope === "persistent" && entry.layer !== "project" ? () => handleDemote(entry.id) : undefined}
												/>
											))}
										</div>
									)}
								</div>
							</ScrollArea>
						</CardContent>
					</Card>
				</div>
			</div>

			<Dialog
				open={dialogOpen}
				onOpenChange={(open) => {
					if (!open) closeModal();
				}}
			>
				<DialogContent className="max-w-4xl w-[96vw] p-0 gap-0 max-h-[90vh] overflow-hidden border-border/60 bg-background/95 shadow-2xl flex flex-col">
					{modalMode === "create" ? (
						<MemoryCreateDialog
							scope={scope}
							onCancel={closeModal}
							onCreatePersistent={handleCreatePersistent}
							onCreateWorking={handleCreateWorking}
						/>
					) : modalMode === "edit" && selectedPersistentEntry ? (
						<MemoryEditDialog
							entry={selectedPersistentEntry}
							onCancel={closeModal}
							onSubmit={(data) => handleUpdatePersistent(selectedPersistentEntry.id, data)}
						/>
					) : selectedEntry ? (
						<MemoryDetailDialog
							entry={selectedEntry}
							persistent={scope === "persistent"}
							onEdit={selectedPersistentEntry ? () => setModalMode("edit") : undefined}
							onPromote={selectedPersistentEntry && selectedPersistentEntry.layer !== "global" ? () => handlePromote(selectedPersistentEntry.id) : undefined}
							onDemote={selectedPersistentEntry && selectedPersistentEntry.layer !== "project" ? () => handleDemote(selectedPersistentEntry.id) : undefined}
							onDelete={() =>
								scope === "persistent"
									? handleDeletePersistent(selectedEntry.id)
									: handleDeleteWorking(selectedEntry.id)
							}
						/>
					) : null}
				</DialogContent>
			</Dialog>
		</>
	);
}

function ScopeButton({
	label,
	description,
	active,
	onClick,
}: {
	label: string;
	description: string;
	active: boolean;
	onClick: () => void;
}) {
	return (
		<button
			type="button"
			onClick={onClick}
			className={cn(
				"rounded-xl border px-3 py-3 text-left transition-colors",
				active
					? "border-primary/40 bg-primary/5 shadow-sm"
					: "border-border/60 bg-background hover:bg-accent/50",
			)}
		>
			<div className="text-sm font-medium text-foreground">{label}</div>
			<div className="mt-1 text-xs text-muted-foreground">{description}</div>
		</button>
	);
}

function FilterChip({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
	return (
		<button
			type="button"
			onClick={onClick}
			className={cn(
				"rounded-full px-3 py-1.5 text-xs font-medium transition-colors",
				active ? "bg-foreground text-background" : "bg-muted text-muted-foreground hover:text-foreground",
			)}
		>
			{children}
		</button>
	);
}

function EmptyGridState({ title, description }: { title: string; description: string }) {
	return (
		<div className="flex min-h-[340px] flex-col items-center justify-center rounded-2xl border border-dashed border-border/60 bg-muted/20 px-6 text-center">
			<Brain className="mb-4 h-10 w-10 text-muted-foreground/50" />
			<p className="text-sm font-medium">{title}</p>
			<p className="mt-2 max-w-md text-sm text-muted-foreground">{description}</p>
		</div>
	);
}

function MemoryGridCard({
	entry,
	onOpen,
	onDelete,
	onPromote,
	onDemote,
}: {
	entry: MemoryEntry;
	onOpen: () => void;
	onDelete: () => void;
	onPromote?: () => void;
	onDemote?: () => void;
}) {
	const preview = entry.content?.replace(/\s+/g, " ").trim();

	return (
		<div
			onClick={onOpen}
			className="group flex min-h-[220px] cursor-pointer flex-col rounded-2xl border border-border/60 bg-card p-4 shadow-sm transition-all hover:-translate-y-0.5 hover:border-border hover:bg-accent/30"
		>
			<div className="mb-3 flex items-start justify-between gap-3">
				<div className="min-w-0 flex-1">
					<div className="flex flex-wrap items-center gap-2">
						<p className="truncate text-sm font-semibold text-foreground">
							{entry.title || "Untitled"}
						</p>
						<LayerBadge layer={entry.layer} />
					</div>
					{entry.category && <div className="mt-2"><CategoryBadge category={entry.category} /></div>}
				</div>
				<div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
					{onPromote && (
						<button
							type="button"
							onClick={(e) => {
								e.stopPropagation();
								onPromote();
							}}
							className="rounded-md p-1.5 hover:bg-accent"
							title="Promote"
						>
							<ChevronUp className="h-3.5 w-3.5" />
						</button>
					)}
					{onDemote && (
						<button
							type="button"
							onClick={(e) => {
								e.stopPropagation();
								onDemote();
							}}
							className="rounded-md p-1.5 hover:bg-accent"
							title="Demote"
						>
							<ChevronDown className="h-3.5 w-3.5" />
						</button>
					)}
					<button
						type="button"
						onClick={(e) => {
							e.stopPropagation();
							onDelete();
						}}
						className="rounded-md p-1.5 text-destructive hover:bg-destructive/10"
						title="Delete"
					>
						<Trash2 className="h-3.5 w-3.5" />
					</button>
				</div>
			</div>

			<div className="flex-1">
				<p className="line-clamp-5 text-sm text-muted-foreground">
					{preview || "No content."}
				</p>
			</div>

			<div className="mt-4 flex items-center justify-between gap-3 border-t border-border/50 pt-3 text-xs text-muted-foreground">
				<span>{formatDate(entry.updatedAt)}</span>
				<span className="truncate font-mono">{entry.id}</span>
			</div>
		</div>
	);
}

function MemoryCreateDialog({
	scope,
	onCancel,
	onCreatePersistent,
	onCreateWorking,
}: {
	scope: MemoryScope;
	onCancel: () => void;
	onCreatePersistent: (data: { title: string; content: string; layer: PersistentMemoryLayer; category: string }) => void;
	onCreateWorking: (data: { title: string; content: string; category: string }) => void;
}) {
	return scope === "persistent" ? (
		<>
			<DialogHeader className="border-b border-border/50 px-6 py-5 text-left">
				<DialogTitle>New memory</DialogTitle>
				<DialogDescription>Save durable knowledge for this project or globally.</DialogDescription>
			</DialogHeader>
			<div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
				<CreatePersistentMemoryForm onSubmit={onCreatePersistent} onCancel={onCancel} />
			</div>
		</>
	) : (
		<>
			<DialogHeader className="border-b border-border/50 px-6 py-5 text-left">
				<DialogTitle>New working memory</DialogTitle>
				<DialogDescription>Capture temporary notes for this session only.</DialogDescription>
			</DialogHeader>
			<div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
				<CreateWorkingMemoryForm onSubmit={onCreateWorking} onCancel={onCancel} />
			</div>
		</>
	);
}

function MemoryEditDialog({
	entry,
	onCancel,
	onSubmit,
}: {
	entry: MemoryEntry;
	onCancel: () => void;
	onSubmit: (data: { title: string; content: string; category: string }) => void;
}) {
	return (
		<>
			<DialogHeader className="border-b border-border/50 px-6 py-5 text-left">
				<DialogTitle>Edit memory</DialogTitle>
				<DialogDescription>Update the title, content, or category.</DialogDescription>
			</DialogHeader>
			<div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
				<EditPersistentMemoryForm entry={entry} onSubmit={onSubmit} onCancel={onCancel} />
			</div>
		</>
	);
}

function MemoryDetailDialog({
	entry,
	persistent,
	onEdit,
	onPromote,
	onDemote,
	onDelete,
}: {
	entry: MemoryEntry;
	persistent: boolean;
	onEdit?: () => void;
	onPromote?: () => void;
	onDemote?: () => void;
	onDelete: () => void;
}) {
	return (
		<>
			<div className="border-b border-border/50 bg-muted/20 px-6 py-5 shrink-0">
				<div className="mb-3 flex flex-wrap items-center gap-2">
					<LayerBadge layer={entry.layer} />
					<CategoryBadge category={entry.category} />
				</div>
				<DialogTitle className="text-2xl font-semibold tracking-tight leading-tight">
					{entry.title || "Untitled"}
				</DialogTitle>
				<DialogDescription className="mt-2 font-mono text-xs">{entry.id}</DialogDescription>
			</div>

			<div className="min-h-0 flex-1 overflow-y-auto bg-background">
				<div className="space-y-6 px-6 py-6">
					<section className="rounded-2xl bg-muted/20 p-5">
						{entry.content ? (
							<MDRender markdown={entry.content} className="prose prose-sm max-w-none dark:prose-invert" />
						) : (
							<p className="text-sm text-muted-foreground">No content.</p>
						)}
					</section>

					{entry.tags && entry.tags.length > 0 && (
						<section>
							<h3 className="mb-2 text-sm font-semibold uppercase tracking-wide text-muted-foreground">Tags</h3>
							<div className="flex flex-wrap gap-2">
								{entry.tags.map((tag) => (
									<span key={tag} className="rounded-full bg-muted px-2.5 py-1 text-xs text-muted-foreground">
										{tag}
									</span>
								))}
							</div>
						</section>
					)}

					<section className="grid gap-4 rounded-2xl border border-border/60 bg-background p-4 sm:grid-cols-2">
						<MetadataItem label="Created" value={formatDate(entry.createdAt, true)} />
						<MetadataItem label="Updated" value={formatDate(entry.updatedAt, true)} />
					</section>
				</div>
			</div>

			<div className="flex flex-wrap justify-end gap-2 border-t border-border/50 bg-muted/10 px-6 py-3 shrink-0">
				{persistent && onEdit && (
					<button
						type="button"
						onClick={onEdit}
						className="inline-flex items-center gap-1.5 rounded-lg border border-border/60 px-3 py-2 text-sm font-medium hover:bg-accent transition-colors"
					>
						<Pencil className="h-4 w-4" />
						Edit
					</button>
				)}
				{persistent && onPromote && (
					<button
						type="button"
						onClick={onPromote}
						className="inline-flex items-center gap-1.5 rounded-lg border border-border/60 px-3 py-2 text-sm font-medium hover:bg-accent transition-colors"
					>
						<ChevronUp className="h-4 w-4" />
						Promote
					</button>
				)}
				{persistent && onDemote && (
					<button
						type="button"
						onClick={onDemote}
						className="inline-flex items-center gap-1.5 rounded-lg border border-border/60 px-3 py-2 text-sm font-medium hover:bg-accent transition-colors"
					>
						<ChevronDown className="h-4 w-4" />
						Demote
					</button>
				)}
				<button
					type="button"
					onClick={onDelete}
					className="inline-flex items-center gap-1.5 rounded-lg border border-destructive/30 px-3 py-2 text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors"
				>
					<Trash2 className="h-4 w-4" />
					Delete
				</button>
			</div>
		</>
	);
}

function MetadataItem({ label, value }: { label: string; value: string }) {
	return (
		<div className="space-y-1">
			<p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
			<p className="text-sm text-foreground">{value}</p>
		</div>
	);
}

function CreatePersistentMemoryForm({
	onSubmit,
	onCancel,
}: {
	onSubmit: (data: { title: string; content: string; layer: PersistentMemoryLayer; category: string }) => void;
	onCancel: () => void;
}) {
	const [title, setTitle] = useState("");
	const [content, setContent] = useState("");
	const [layer, setLayer] = useState<PersistentMemoryLayer>("project");
	const [category, setCategory] = useState("");

	return (
		<BaseForm
			onCancel={onCancel}
			onSubmit={() => onSubmit({ title, content, layer, category })}
			submitDisabled={!content.trim()}
			titleValue={title}
			onTitleChange={setTitle}
			contentValue={content}
			onContentChange={setContent}
			categoryValue={category}
			onCategoryChange={setCategory}
			extraFields={
				<div className="space-y-2">
					<label className="text-sm font-medium text-foreground">Layer</label>
					<select
						value={layer}
						onChange={(e) => setLayer(e.target.value as PersistentMemoryLayer)}
						className="w-full rounded-xl border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-primary"
					>
						<option value="project">Project</option>
						<option value="global">Global</option>
					</select>
				</div>
			}
		/>
	);
}

function CreateWorkingMemoryForm({
	onSubmit,
	onCancel,
}: {
	onSubmit: (data: { title: string; content: string; category: string }) => void;
	onCancel: () => void;
}) {
	const [title, setTitle] = useState("");
	const [content, setContent] = useState("");
	const [category, setCategory] = useState("");

	return (
		<BaseForm
			onCancel={onCancel}
			onSubmit={() => onSubmit({ title, content, category })}
			submitDisabled={!content.trim()}
			titleValue={title}
			onTitleChange={setTitle}
			contentValue={content}
			onContentChange={setContent}
			categoryValue={category}
			onCategoryChange={setCategory}
		/>
	);
}

function EditPersistentMemoryForm({
	entry,
	onSubmit,
	onCancel,
}: {
	entry: MemoryEntry;
	onSubmit: (data: { title: string; content: string; category: string }) => void;
	onCancel: () => void;
}) {
	const [title, setTitle] = useState(entry.title || "");
	const [content, setContent] = useState(entry.content || "");
	const [category, setCategory] = useState(entry.category || "");

	return (
		<BaseForm
			onCancel={onCancel}
			onSubmit={() => onSubmit({ title, content, category })}
			submitLabel="Save changes"
			submitDisabled={!content.trim()}
			titleValue={title}
			onTitleChange={setTitle}
			contentValue={content}
			onContentChange={setContent}
			categoryValue={category}
			onCategoryChange={setCategory}
		/>
	);
}

function BaseForm({
	onCancel,
	onSubmit,
	submitDisabled,
	submitLabel = "Create",
	titleValue,
	onTitleChange,
	contentValue,
	onContentChange,
	categoryValue,
	onCategoryChange,
	extraFields,
}: {
	onCancel: () => void;
	onSubmit: () => void;
	submitDisabled: boolean;
	submitLabel?: string;
	titleValue: string;
	onTitleChange: (value: string) => void;
	contentValue: string;
	onContentChange: (value: string) => void;
	categoryValue: string;
	onCategoryChange: (value: string) => void;
	extraFields?: ReactNode;
}) {
	return (
		<form
			onSubmit={(e) => {
				e.preventDefault();
				onSubmit();
			}}
			className="space-y-5"
		>
			<div className="space-y-2">
				<label className="text-sm font-medium text-foreground">Title</label>
				<input
					type="text"
					placeholder="Optional title"
					value={titleValue}
					onChange={(e) => onTitleChange(e.target.value)}
					className="w-full rounded-xl border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-primary"
				/>
			</div>

			<div className="space-y-2">
				<label className="text-sm font-medium text-foreground">Content</label>
				<textarea
					placeholder="Write in markdown"
					value={contentValue}
					onChange={(e) => onContentChange(e.target.value)}
					rows={10}
					className="w-full resize-y rounded-xl border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-primary"
				/>
			</div>

			<div className="grid gap-5 md:grid-cols-2">
				<div className="space-y-2">
					<label className="text-sm font-medium text-foreground">Category</label>
					<input
						type="text"
						placeholder="Optional category"
						value={categoryValue}
						onChange={(e) => onCategoryChange(e.target.value)}
						className="w-full rounded-xl border border-border/60 bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-primary"
					/>
				</div>
				{extraFields}
			</div>

			<div className="flex flex-wrap justify-end gap-2 border-t border-border/50 pt-5">
				<button
					type="button"
					onClick={onCancel}
					className="rounded-lg px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
				>
					Cancel
				</button>
				<button
					type="submit"
					disabled={submitDisabled}
					className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
				>
					{submitLabel}
				</button>
			</div>
		</form>
	);
}
