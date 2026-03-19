import { useEffect, useState, useRef, useCallback } from "react";
import {
	Settings,
	Check,
	Plus,
	Trash2,
	Columns3,
	User,
	Clock,
	Tag,
	Palette,
	Eye,
	Terminal,
	Bot,
	Loader2,
	AlertCircle,
	CheckCircle2,
	Download,
	Package,
	ChevronRight,
	RefreshCw,
	X,
	type LucideIcon,
} from "lucide-react";
import { ScrollArea } from "../components/ui/ScrollArea";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Switch } from "../components/ui/switch";
import { Separator } from "../components/ui/separator";
import { Label } from "../components/ui/label";
import { useConfig, type Config } from "../contexts/ConfigContext";
import { useOpenCode } from "../contexts/OpenCodeContext";
import { useOpenCodeModelManager } from "../hooks/useOpencodeModelManager";
import { OpenCodeModelManager } from "../components/organisms/OpenCodeModelManager";
import { toast } from "../components/ui/sonner";
import { importApi, type Import, type ImportDetail, type ImportResult } from "../api/client";

const DEFAULT_STATUSES = ["todo", "in-progress", "in-review", "done", "blocked", "on-hold", "urgent"];
const COLOR_OPTIONS = ["gray", "blue", "green", "yellow", "red", "purple", "orange", "pink", "cyan", "indigo"];

// ── Category definitions ──────────────────────────────────────────

type Category = "general" | "board" | "ai";

interface CategoryDef {
	id: Category;
	label: string;
	icon: LucideIcon;
	description: string;
}

const CATEGORIES: CategoryDef[] = [
	{ id: "general", label: "General", icon: Settings, description: "Project name, defaults, and preferences" },
	{ id: "board", label: "Board", icon: Columns3, description: "Kanban statuses, colors, and visible columns" },
	{ id: "ai", label: "AI", icon: Bot, description: "OpenCode connection used by Chat UI" },
];

// ── Auto-save hook ────────────────────────────────────────────────

function useAutoSave(config: Config, updateConfig: (c: Partial<Config>) => Promise<void>, initialized: boolean) {
	const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const prevRef = useRef<string>("");

	const save = useCallback(
		(cfg: Config) => {
			if (timerRef.current) clearTimeout(timerRef.current);
			timerRef.current = setTimeout(async () => {
				const serialized = JSON.stringify(cfg);
				if (serialized === prevRef.current) return;
				prevRef.current = serialized;
				try {
					await updateConfig(cfg);
					toast.success("Saved", { duration: 1500, position: "bottom-right" });
				} catch {
					toast.error("Failed to save settings", { position: "bottom-right" });
				}
			}, 600);
		},
		[updateConfig],
	);

	// Set initial snapshot so first render doesn't trigger save
	useEffect(() => {
		if (initialized) {
			prevRef.current = JSON.stringify(config);
		}
	}, [initialized]); // eslint-disable-line react-hooks/exhaustive-deps

	return save;
}

// ── Section header component ──────────────────────────────────────

function SectionHeader({ icon: Icon, title, description }: { icon: LucideIcon; title: string; description: string }) {
	return (
		<div className="mb-5">
			<div className="flex items-center gap-2.5 mb-1">
				<Icon className="w-[18px] h-[18px] text-muted-foreground" />
				<h3 className="text-sm font-semibold">{title}</h3>
			</div>
			<p className="text-xs text-muted-foreground ml-[30px]">{description}</p>
		</div>
	);
}

// ── Field row component ───────────────────────────────────────────

function FieldRow({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
	return (
		<div className="flex items-start gap-4 py-3">
			<div className="w-44 shrink-0 pt-2">
				<div className="text-sm font-medium">{label}</div>
				{hint && <div className="text-xs text-muted-foreground mt-0.5">{hint}</div>}
			</div>
			<div className="flex-1 min-w-0">{children}</div>
		</div>
	);
}

// ── Main component ────────────────────────────────────────────────

export default function ConfigPage() {
	const { config: globalConfig, loading, updateConfig, chatUIEnabled } = useConfig();
	const [config, setConfig] = useState<Config>({});
	const [activeCategory, setActiveCategory] = useState<Category>("general");
	const [viewMode, setViewMode] = useState<"form" | "json">("form");
	const [jsonText, setJsonText] = useState("");
	const [jsonError, setJsonError] = useState<string | null>(null);
	const [newStatus, setNewStatus] = useState("");
	const [initialized, setInitialized] = useState(false);
	const [saving, setSaving] = useState(false);
	const { status: openCodeStatus, statusLoading: openCodeStatusLoading, providerResponse, providersLoading, lastLoadedAt, refreshAll } =
		useOpenCode();

	// Imports state
	const [imports, setImports] = useState<Import[]>([]);
	const [importsLoading, setImportsLoading] = useState(true);
	const [selectedImport, setSelectedImport] = useState<ImportDetail | null>(null);
	const [showAddModal, setShowAddModal] = useState(false);
	const [showRemoveConfirm, setShowRemoveConfirm] = useState(false);
	const [syncingImport, setSyncingImport] = useState<string | null>(null);
	const [removingImport, setRemovingImport] = useState<string | null>(null);
	const [removeDeleteFiles, setRemoveDeleteFiles] = useState(false);

	// Add import form state
	const [addSource, setAddSource] = useState("");
	const [addName, setAddName] = useState("");
	const [addType, setAddType] = useState("");
	const [addRef, setAddRef] = useState("");
	const [addLink, setAddLink] = useState(false);
	const [addDryRun, setAddDryRun] = useState(true);
	const [adding, setAdding] = useState(false);
	const [addResult, setAddResult] = useState<ImportResult | null>(null);
	const [addError, setAddError] = useState<string | null>(null);

	// Load imports
	const loadImports = useCallback(async () => {
		try {
			const data = await importApi.list();
			setImports(data.imports);
		} catch (err) {
			console.error("Failed to load imports:", err);
		} finally {
			setImportsLoading(false);
		}
	}, []);

	// Load import detail
	const loadImportDetail = async (name: string) => {
		try {
			const data = await importApi.get(name);
			setSelectedImport(data.import);
		} catch (err) {
			console.error("Failed to load import:", err);
		}
	};

	// Handle add import
	const handleAddImport = async () => {
		if (!addSource.trim()) return;
		setAdding(true);
		setAddError(null);
		setAddResult(null);
		try {
			const result = await importApi.add({
				source: addSource,
				name: addName || undefined,
				type: addType || undefined,
				ref: addRef || undefined,
				link: addLink,
				dryRun: addDryRun,
			});
			setAddResult(result);
			if (!addDryRun) {
				loadImports();
				setShowAddModal(false);
				resetAddForm();
			}
		} catch (err) {
			setAddError(err instanceof Error ? err.message : String(err));
		} finally {
			setAdding(false);
		}
	};

	// Handle sync
	const handleSync = async (name: string) => {
		setSyncingImport(name);
		try {
			await importApi.sync(name);
			loadImports();
			toast.success(`Synced "${name}"`);
		} catch (err) {
			console.error("Failed to sync:", err);
			toast.error(`Failed to sync "${name}"`);
		} finally {
			setSyncingImport(null);
		}
	};

	// Handle remove
	const handleRemove = async () => {
		if (!selectedImport) return;
		setRemovingImport(selectedImport.name);
		try {
			await importApi.remove(selectedImport.name, removeDeleteFiles);
			setShowRemoveConfirm(false);
			setSelectedImport(null);
			loadImports();
		} catch (err) {
			console.error("Failed to remove:", err);
		} finally {
			setRemovingImport(null);
		}
	};

	// Reset add form
	const resetAddForm = () => {
		setAddSource("");
		setAddName("");
		setAddType("");
		setAddRef("");
		setAddLink(false);
		setAddDryRun(true);
		setAddResult(null);
		setAddError(null);
	};

	// Close detail view
	const closeImportDetail = () => {
		setSelectedImport(null);
	};

	useEffect(() => {
		loadImports();
	}, [loadImports]);

	const autoSave = useAutoSave(config, updateConfig, initialized);

	// Initialize from global config
	useEffect(() => {
		if (!loading && !initialized) {
			setConfig(globalConfig);
			setJsonText(JSON.stringify(globalConfig, null, 2));
			setInitialized(true);
		}
	}, [globalConfig, loading, initialized]);

	useEffect(() => {
		if (initialized) {
			void refreshAll({ silent: true });
		}
	}, [initialized, refreshAll]);

	// Sync JSON text when switching to JSON mode
	useEffect(() => {
		if (viewMode === "json") {
			setJsonText(JSON.stringify(config, null, 2));
		}
	}, [viewMode]); // eslint-disable-line react-hooks/exhaustive-deps

	// Update helper — updates local state + triggers auto-save
	const update = useCallback(
		(patch: Partial<Config>) => {
			setConfig((prev) => {
				const next = { ...prev, ...patch };
				autoSave(next);
				return next;
			});
		},
		[autoSave],
	);

	const handleAddStatus = () => {
		if (!newStatus.trim()) return;
		const statusKey = newStatus.toLowerCase().replace(/\s+/g, "-");
		const currentStatuses = config.statuses || DEFAULT_STATUSES;
		if (currentStatuses.includes(statusKey)) {
			toast.error("Status already exists");
			return;
		}
		update({
			statuses: [...currentStatuses, statusKey],
			statusColors: { ...(config.statusColors || {}), [statusKey]: "gray" },
		});
		setNewStatus("");
	};

	const handleRemoveStatus = (status: string) => {
		const currentStatuses = config.statuses || DEFAULT_STATUSES;
		const newColors = { ...(config.statusColors || {}) };
		delete newColors[status];
		update({
			statuses: currentStatuses.filter((s) => s !== status),
			statusColors: newColors,
			visibleColumns: (config.visibleColumns || []).filter((c) => c !== status),
		});
	};

	const handleJsonSave = async () => {
		setSaving(true);
		try {
			const parsed = JSON.parse(jsonText);
			setConfig(parsed);
			setJsonError(null);
			await updateConfig(parsed);
			toast.success("Saved", { duration: 1500, position: "bottom-right" });
		} catch (e) {
			if (e instanceof SyntaxError) {
				setJsonError("Invalid JSON syntax");
			} else {
				toast.error("Failed to save");
			}
		} finally {
			setSaving(false);
		}
	};

	const updateOpenCodeServer = useCallback(
		(patch: NonNullable<Config["opencodeServer"]>) => {
			update({
				opencodeServer: {
					...(config.opencodeServer || {}),
					...patch,
				},
			});
		},
		[config.opencodeServer, update],
	);

	const { modelCatalog, updateModelPref, toggleProviderHidden, setDefaultModel } = useOpenCodeModelManager({
		settings: config.opencodeModels,
		providerResponse,
		status: openCodeStatus,
		lastLoadedAt,
		onChange: (nextSettings) => update({ opencodeModels: nextSettings }),
	});

	if (loading) {
		return (
			<div className="p-6 flex items-center justify-center h-64">
				<div className="text-lg text-muted-foreground">Loading configuration...</div>
			</div>
		);
	}

	const statuses = config.statuses || DEFAULT_STATUSES;
	const statusColors = config.statusColors || {};

	// ── Render category content ───────────────────────────────────

	const renderGeneral = () => (
		<div>
			<SectionHeader icon={Settings} title="Project" description="Basic project information" />

			<FieldRow label="Project name" hint="Display name for your project">
				<Input
					value={config.name || ""}
					onChange={(e) => update({ name: e.target.value })}
					placeholder="My Project"
				/>
			</FieldRow>

			<Separator className="my-1" />

			<SectionHeader icon={User} title="Defaults" description="Default values for new tasks" />

			<FieldRow label="Assignee" hint="Default assignee for new tasks">
				<Input
					value={config.defaultAssignee || ""}
					onChange={(e) => update({ defaultAssignee: e.target.value })}
					placeholder="@username"
				/>
			</FieldRow>

			<FieldRow label="Priority">
				<select
					value={config.defaultPriority || "medium"}
					onChange={(e) => update({ defaultPriority: e.target.value as Config["defaultPriority"] })}
					className="w-full px-3 py-2 rounded-md border bg-input text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				>
					<option value="low">Low</option>
					<option value="medium">Medium</option>
					<option value="high">High</option>
				</select>
			</FieldRow>

			<FieldRow label="Labels" hint="Comma-separated">
				<Input
					value={config.defaultLabels?.join(", ") || ""}
					onChange={(e) =>
						update({
							defaultLabels: e.target.value
								.split(",")
								.map((l) => l.trim())
								.filter(Boolean),
						})
					}
					placeholder="frontend, backend, ui"
				/>
			</FieldRow>

			<Separator className="my-1" />

			<SectionHeader icon={Download} title="Imports" description="Imported templates and docs" />

			{importsLoading ? (
				<div className="flex items-center justify-center py-6">
					<Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
				</div>
			) : imports.length === 0 ? (
				<div className="py-4 text-center border rounded-lg bg-muted/20">
					<Download className="w-8 h-8 mx-auto text-muted-foreground" />
					<p className="mt-2 text-sm text-muted-foreground">No imports yet</p>
					<Button onClick={() => setShowAddModal(true)} variant="outline" size="sm" className="mt-2">
						<Plus className="w-4 h-4 mr-2" />
						Add Import
					</Button>
				</div>
			) : (
				<div className="space-y-2 mb-4">
					{imports.map((imp) => (
						<div
							key={imp.name}
							className="flex items-center justify-between p-3 rounded-lg border bg-card hover:bg-accent/50 transition-colors cursor-pointer"
							onClick={() => loadImportDetail(imp.name)}
						>
							<div className="flex items-center gap-3 min-w-0">
								<Package className="w-4 h-4 text-muted-foreground shrink-0" />
								<div className="min-w-0">
									<div className="text-sm font-medium truncate">{imp.name}</div>
									<div className="text-xs text-muted-foreground truncate">{imp.source}</div>
								</div>
							</div>
							<div className="flex items-center gap-2">
								<span className="text-xs text-muted-foreground">{imp.fileCount} files</span>
								<ChevronRight className="w-4 h-4 text-muted-foreground" />
							</div>
						</div>
					))}
				</div>
			)}

			{imports.length > 0 && (
				<div className="flex justify-end mb-4">
					<Button onClick={() => setShowAddModal(true)} size="sm" variant="outline">
						<Plus className="w-4 h-4 mr-2" />
						Add
					</Button>
				</div>
			)}

			<Separator className="my-1" />

			<SectionHeader icon={Clock} title="Preferences" description="Display and editor settings" />

			<FieldRow label="Time format">
				<select
					value={config.timeFormat || "24h"}
					onChange={(e) => update({ timeFormat: e.target.value as Config["timeFormat"] })}
					className="w-full px-3 py-2 rounded-md border bg-input text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				>
					<option value="12h">12-hour (AM/PM)</option>
					<option value="24h">24-hour</option>
				</select>
			</FieldRow>

			<FieldRow label="Editor" hint="CLI editor command">
				<Input
					value={config.editor || ""}
					onChange={(e) => update({ editor: e.target.value })}
					placeholder="code, vim, nano"
				/>
			</FieldRow>
		</div>
	);

	const renderBoard = () => (
		<div>
			<SectionHeader icon={Tag} title="Task Statuses" description="Define statuses and their colors for the Kanban board" />

			<div className="space-y-1.5 mb-4">
				{statuses.map((status) => (
					<div key={status} className="flex items-center gap-3 px-3 py-2 rounded-md bg-accent/50 hover:bg-accent transition-colors">
						<Palette className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
						<span className="flex-1 font-mono text-sm">{status}</span>
						<select
							value={statusColors[status] || "gray"}
							onChange={(e) => update({ statusColors: { ...(config.statusColors || {}), [status]: e.target.value } })}
							className="px-2 py-1 rounded border bg-input text-xs focus:outline-none focus:ring-2 focus:ring-ring"
						>
							{COLOR_OPTIONS.map((color) => (
								<option key={color} value={color}>{color}</option>
							))}
						</select>
						<button
							type="button"
							onClick={() => handleRemoveStatus(status)}
							className="p-1 text-muted-foreground hover:text-destructive rounded transition-colors"
						>
							<Trash2 className="w-3.5 h-3.5" />
						</button>
					</div>
				))}
			</div>

			<div className="flex gap-2 mb-6">
				<Input
					value={newStatus}
					onChange={(e) => setNewStatus(e.target.value)}
					placeholder="new-status"
					className="flex-1"
					onKeyDown={(e) => e.key === "Enter" && handleAddStatus()}
				/>
				<Button size="sm" onClick={handleAddStatus} variant="outline">
					<Plus className="w-4 h-4 mr-1" />
					Add
				</Button>
			</div>

			<Separator className="my-4" />

			<SectionHeader icon={Eye} title="Visible Columns" description="Choose which columns appear on the Kanban board" />

			<div className="space-y-1">
				{statuses.map((column) => {
					const isVisible = config.visibleColumns?.includes(column) ?? true;
					const label = column
						.split("-")
						.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
						.join(" ");

					return (
						<div
							key={column}
							className="flex items-center justify-between px-3 py-2.5 rounded-md hover:bg-accent/50 transition-colors"
						>
							<span className="text-sm">{label}</span>
							<Switch
								checked={isVisible}
								onCheckedChange={(checked) => {
									const current = config.visibleColumns || statuses;
									const updated = checked
										? [...current, column]
										: current.filter((c) => c !== column);
									update({ visibleColumns: updated });
								}}
							/>
						</div>
					);
				})}
			</div>
		</div>
	);

	const renderAI = () => {
		const statusTone = openCodeStatusLoading
			? "border-border bg-muted/40 text-muted-foreground"
			: openCodeStatus?.available
				? "border-emerald-200 bg-emerald-50 text-emerald-700"
				: "border-amber-200 bg-amber-50 text-amber-700";

		return (
			<div>
				<SectionHeader icon={Bot} title="OpenCode" description="Configure the OpenCode server used by Chat UI" />

				<FieldRow label="Connection" hint="Chat UI is blocked when OpenCode is unavailable">
					<div className="space-y-3">
						<div className={`flex items-start gap-2 rounded-md border px-3 py-2 text-sm ${statusTone}`}>
							{openCodeStatusLoading ? (
								<Loader2 className="mt-0.5 h-4 w-4 shrink-0 animate-spin" />
							) : openCodeStatus?.available ? (
								<CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
							) : (
								<AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
							)}
							<div className="min-w-0">
								<div className="font-medium">
									{openCodeStatusLoading
										? "Checking OpenCode..."
										: openCodeStatus?.available
											? `Connected to ${openCodeStatus.host}:${openCodeStatus.port}`
											: openCodeStatus?.error || "OpenCode is unavailable."}
								</div>
								{!openCodeStatusLoading && !openCodeStatus?.cliAvailable && (
									<div className="mt-1 text-xs opacity-80">
										`opencode` CLI was not found, so auto-start is unavailable.
									</div>
								)}
							</div>
						</div>
						<Button variant="outline" size="sm" onClick={() => void refreshAll()} disabled={openCodeStatusLoading || providersLoading}>
							{openCodeStatusLoading || providersLoading ? "Checking..." : "Refresh status"}
						</Button>
					</div>
				</FieldRow>

				<FieldRow label="Password" hint="Optional basic auth password">
					<Input
						type="password"
						value={config.opencodeServer?.password || ""}
						onChange={(e) => updateOpenCodeServer({ password: e.target.value })}
						placeholder="Leave empty if OpenCode has no password"
					/>
				</FieldRow>

				<Separator className="my-4" />

				<SectionHeader
					icon={Bot}
					title="Model Manager"
					description="Enable models, choose the project default, and control what appears in the chat picker"
				/>

				<FieldRow label="Catalog" hint="Live provider/model catalog from OpenCode">
					<OpenCodeModelManager
						catalog={modelCatalog}
						lastLoadedAt={lastLoadedAt}
						onSetDefaultModel={setDefaultModel}
						onUpdateModelPref={updateModelPref}
						onToggleProviderHidden={toggleProviderHidden}
						showProviderVisibility
					/>
				</FieldRow>
			</div>
		);
	};

	const contentByCategory: Record<Category, () => React.ReactNode> = {
		general: renderGeneral,
		board: renderBoard,
		ai: renderAI,
	};

	// ── JSON mode ─────────────────────────────────────────────────

	const renderJsonMode = () => (
		<div className="space-y-4">
			<div>
				<label className="block text-sm font-medium mb-2 flex items-center gap-2">
					<Terminal className="w-4 h-4 text-muted-foreground" />
					config.json
				</label>
				<textarea
					value={jsonText}
					onChange={(e) => {
						setJsonText(e.target.value);
						setJsonError(null);
					}}
					className="w-full h-[calc(100vh-280px)] min-h-[300px] px-4 py-3 rounded-lg border bg-input font-mono text-sm focus:outline-none focus:ring-2 focus:ring-ring resize-none"
					spellCheck={false}
				/>
				{jsonError && <p className="mt-2 text-sm text-destructive">{jsonError}</p>}
			</div>
			<Button onClick={handleJsonSave} disabled={saving}>
				<Check className="w-4 h-4 mr-2" />
				{saving ? "Saving..." : "Save"}
			</Button>
		</div>
	);

	// ── Main layout ───────────────────────────────────────────────

	return (
		<div className="h-full flex flex-col">
			{/* Top bar */}
			<div className="px-6 py-4 border-b shrink-0 flex items-center justify-between">
				<h1 className="text-lg font-semibold">Settings</h1>
				<div className="flex rounded-md border overflow-hidden">
					<button
						type="button"
						onClick={() => setViewMode("form")}
						className={`px-3 py-1 text-xs font-medium transition-colors ${
							viewMode === "form"
								? "bg-primary text-primary-foreground"
								: "bg-secondary text-secondary-foreground hover:bg-secondary/80"
						}`}
					>
						Form
					</button>
					<button
						type="button"
						onClick={() => setViewMode("json")}
						className={`px-3 py-1 text-xs font-medium transition-colors ${
							viewMode === "json"
								? "bg-primary text-primary-foreground"
								: "bg-secondary text-secondary-foreground hover:bg-secondary/80"
						}`}
					>
						JSON
					</button>
				</div>
			</div>

			{viewMode === "json" ? (
				<ScrollArea className="flex-1">
					<div className="p-6 max-w-3xl">{renderJsonMode()}</div>
				</ScrollArea>
			) : (
				<div className="flex-1 flex min-h-0">
					{/* Sidebar */}
					<nav className="w-52 shrink-0 border-r bg-accent/30 p-3 hidden md:block">
						<div className="space-y-0.5">
							{CATEGORIES.filter((cat) => cat.id !== "ai" || chatUIEnabled).map((cat) => {
								const Icon = cat.icon;
								const isActive = activeCategory === cat.id;
								return (
									<button
										key={cat.id}
										type="button"
										onClick={() => setActiveCategory(cat.id)}
										className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-md text-sm transition-colors text-left ${
											isActive
												? "bg-accent font-medium"
												: "text-muted-foreground hover:bg-accent/60 hover:text-foreground"
										}`}
									>
										<Icon className="w-4 h-4 shrink-0" />
										{cat.label}
									</button>
								);
							})}
						</div>
					</nav>

					{/* Mobile tabs (visible on small screens) */}
					<div className="md:hidden border-b px-4 pt-2 flex gap-1 shrink-0">
						{CATEGORIES.filter((cat) => cat.id !== "ai" || chatUIEnabled).map((cat) => {
							const Icon = cat.icon;
							const isActive = activeCategory === cat.id;
							return (
								<button
									key={cat.id}
									type="button"
									onClick={() => setActiveCategory(cat.id)}
									className={`flex items-center gap-1.5 px-3 py-2 text-xs font-medium rounded-t-md transition-colors ${
										isActive
											? "bg-background border border-b-0 text-foreground"
											: "text-muted-foreground hover:text-foreground"
									}`}
								>
									<Icon className="w-3.5 h-3.5" />
									{cat.label}
								</button>
							);
						})}
					</div>

					{/* Content */}
					<ScrollArea className="flex-1">
						<div className="p-6 max-w-2xl">
							{contentByCategory[activeCategory]()}
						</div>
					</ScrollArea>
				</div>
			)}

			{/* Add Import Modal */}
			{showAddModal && (
				<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
					<div className="bg-card rounded-lg shadow-xl max-w-lg w-full max-h-[90vh] overflow-y-auto">
						<div className="p-6 border-b">
							<h2 className="text-lg font-semibold">Add Import</h2>
							<p className="text-sm text-muted-foreground mt-1">
								Import templates and docs from an external source
							</p>
						</div>

						<div className="p-6 space-y-4">
							<div>
								<Label className="mb-2 block">
									Source <span className="text-red-500">*</span>
								</Label>
								<Input
									type="text"
									value={addSource}
									onChange={(e) => setAddSource(e.target.value)}
									placeholder="https://github.com/org/repo.git, @org/package, or ../path"
								/>
							</div>

							<div className="grid grid-cols-2 gap-4">
								<div>
									<Label className="mb-2 block">Name (optional)</Label>
									<Input
										type="text"
										value={addName}
										onChange={(e) => setAddName(e.target.value)}
										placeholder="Auto-detect"
									/>
								</div>
								<div>
									<Label className="mb-2 block">Type (optional)</Label>
									<select
										value={addType}
										onChange={(e) => setAddType(e.target.value)}
										className="w-full px-3 py-2 rounded-lg border border-border/40 bg-background"
									>
										<option value="">Auto-detect</option>
										<option value="git">Git</option>
										<option value="npm">NPM</option>
										<option value="local">Local</option>
									</select>
								</div>
							</div>

							<div>
								<Label className="mb-2 block">Ref (optional)</Label>
								<Input
									type="text"
									value={addRef}
									onChange={(e) => setAddRef(e.target.value)}
									placeholder="Branch, tag, or version"
								/>
							</div>

							<div className="flex items-center gap-4 p-3 rounded-lg border border-border/40 bg-muted/30">
								<Switch id="link" checked={addLink} onCheckedChange={setAddLink} />
								<Label htmlFor="link" className="text-sm cursor-pointer flex-1">
									<span className="font-medium">Symlink</span>
									<span className="text-muted-foreground ml-2">(local only)</span>
								</Label>
							</div>

							<div className="flex items-center gap-4 p-3 rounded-lg border border-border/40 bg-muted/30">
								<Switch id="dry-run" checked={addDryRun} onCheckedChange={setAddDryRun} />
								<Label htmlFor="dry-run" className="text-sm cursor-pointer flex-1">
									<span className="font-medium">Preview mode</span>
									<span className="text-muted-foreground ml-2">(no files created)</span>
								</Label>
							</div>

							{addError && (
								<div className="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 p-4">
									<div className="flex items-start gap-2 text-red-600 dark:text-red-400">
										<AlertCircle className="w-5 h-5 shrink-0 mt-0.5" />
										<div className="text-sm">{addError}</div>
									</div>
								</div>
							)}

							{addResult && (
								<div className={`border rounded-lg p-4 ${addResult.dryRun ? "bg-blue-50 dark:bg-blue-950/30 border-blue-200" : "bg-green-50 dark:bg-green-950/30 border-green-200"}`}>
									<div className="font-medium">
										{addResult.dryRun ? "Preview Complete" : "Import Complete"}
									</div>
									{addResult.summary && (
										<div className="text-sm text-muted-foreground mt-1">
											{addResult.summary.added} added, {addResult.summary.updated} updated, {addResult.summary.skipped} skipped
										</div>
									)}
									{addResult.dryRun && (
										<Button onClick={() => { setAddDryRun(false); handleAddImport(); }} className="mt-3" size="sm">
											Import Now
										</Button>
									)}
								</div>
							)}
						</div>

						<div className="p-6 border-t flex justify-end gap-3">
							<Button variant="secondary" onClick={() => { resetAddForm(); setShowAddModal(false); }} disabled={adding}>
								Cancel
							</Button>
							{(!addResult || addResult.dryRun) && (
								<Button onClick={handleAddImport} disabled={adding || !addSource.trim()}>
									{adding ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : addDryRun ? "Preview" : "Import"}
								</Button>
							)}
						</div>
					</div>
				</div>
			)}

			{/* Import Detail Modal */}
			{selectedImport && (
				<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
					<div className="bg-card rounded-lg shadow-xl max-w-lg w-full max-h-[90vh] overflow-y-auto">
						<div className="p-6 border-b">
							<div className="flex items-center justify-between">
								<div className="flex items-center gap-3">
									<Package className="w-6 h-6 text-muted-foreground" />
									<div>
										<h2 className="text-lg font-semibold">{selectedImport.name}</h2>
										<p className="text-sm text-muted-foreground font-mono">{selectedImport.source}</p>
									</div>
								</div>
								<Button variant="ghost" size="sm" onClick={closeImportDetail}>
									<X className="w-4 h-4" />
								</Button>
							</div>
						</div>

						<div className="p-6 space-y-4">
							<div className="grid grid-cols-2 gap-4">
								<div>
									<div className="text-xs text-muted-foreground mb-1">Type</div>
									<div className="font-medium capitalize">{selectedImport.type}</div>
								</div>
								<div>
									<div className="text-xs text-muted-foreground mb-1">Files</div>
									<div className="font-medium">{selectedImport.fileCount} files</div>
								</div>
							</div>

							<div className="flex gap-2 pt-2">
								<Button onClick={() => handleSync(selectedImport.name)} disabled={syncingImport === selectedImport.name} variant="outline" size="sm">
									{syncingImport === selectedImport.name ? <Loader2 className="w-4 h-4 animate-spin mr-2" /> : <RefreshCw className="w-4 h-4 mr-2" />}
									Sync
								</Button>
								<Button onClick={() => setShowRemoveConfirm(true)} variant="outline" size="sm" className="text-red-600 hover:text-red-700">
									<Trash2 className="w-4 h-4 mr-2" />
									Remove
								</Button>
							</div>
						</div>
					</div>
				</div>
			)}

			{/* Remove Confirmation Modal */}
			{showRemoveConfirm && selectedImport && (
				<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
					<div className="bg-card rounded-lg shadow-xl max-w-md w-full">
						<div className="p-6 border-b">
							<h2 className="text-lg font-semibold text-red-600">Remove Import</h2>
						</div>

						<div className="p-6 space-y-4">
							<p>Are you sure you want to remove the import <strong>{selectedImport.name}</strong>?</p>

							<div className="flex items-center gap-4 p-3 rounded-lg border border-border/40 bg-muted/30">
								<Switch id="delete-files" checked={removeDeleteFiles} onCheckedChange={setRemoveDeleteFiles} />
								<Label htmlFor="delete-files" className="text-sm cursor-pointer flex-1">
									<span className="font-medium">Also delete imported files</span>
									<span className="text-muted-foreground block text-xs mt-0.5">
										{selectedImport.fileCount} files will be permanently deleted
									</span>
								</Label>
							</div>
						</div>

						<div className="p-6 border-t flex justify-end gap-3">
							<Button variant="secondary" onClick={() => { setShowRemoveConfirm(false); setRemoveDeleteFiles(false); }} disabled={removingImport !== null}>
								Cancel
							</Button>
							<Button onClick={handleRemove} disabled={removingImport !== null} className="bg-red-600 hover:bg-red-700 text-white">
								{removingImport ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : null}
								Remove
							</Button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
