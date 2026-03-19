import { useCallback, useEffect, useRef, useState } from "react";
import { useRouterState } from "@tanstack/react-router";
import {
	Plus,
	FileText,
	Pencil,
	Check,
	X,
	Copy,
	ListChecks,
	ClipboardCheck,
	ChevronDown,
	ChevronUp,
	ExternalLink,
	ArrowLeft,
	Maximize2,
	Minimize2,
	Menu,
} from "lucide-react";
import { MDEditor, MDRender } from "../components/editor";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Progress } from "../components/ui/progress";
import { createDoc, updateDoc } from "../api/client";
import { useGlobalTask } from "../contexts/GlobalTaskContext";
import { useDocs } from "../contexts/DocsContext";
import { DocsFileManager } from "../components/organisms/DocsFileManager";
import {
	toDisplayPath,
	normalizePathForAPI,
	isSpec,
	getSpecStatus,
	parseACProgress,
} from "../lib/utils";
import { navigateTo } from "../lib/navigation";
import { DocsTOC } from "../components/molecules/DocsTOC";
import { TaskPreviewDialog } from "../components/organisms/TaskDetail/TaskPreviewDialog";
import { Sheet, SheetContent, SheetTitle } from "../components/ui/sheet";

export default function DocsPage() {
	const location = useRouterState({ select: (state) => state.location });
	const { openTask } = useGlobalTask();
	const {
		docs,
		loading,
		error,
		selectedDoc,
		setSelectedDoc,
		isEditing,
		setIsEditing,
		editedContent,
		setEditedContent,
		linkedTasks,
		showSpecsOnly,
		setShowSpecsOnly,
		linkedTasksExpanded,
		setLinkedTasksExpanded,
		loadDocs,
		currentFolder,
		navigateToFolder,
	} = useDocs();

	const [saving, setSaving] = useState(false);
	const [showCreateView, setShowCreateView] = useState(false);
	const [newDocTitle, setNewDocTitle] = useState("");
	const [newDocDescription, setNewDocDescription] = useState("");
	const [newDocTags, setNewDocTags] = useState("");
	const [newDocFolder, setNewDocFolder] = useState(currentFolder || "");
	const [newDocContent, setNewDocContent] = useState("");
	const [creating, setCreating] = useState(false);
	const [pathCopied, setPathCopied] = useState(false);
	const [previewTaskId, setPreviewTaskId] = useState<string | null>(null);
	const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
	const [docSearchQuery, setDocSearchQuery] = useState("");
	const [wideMode, setWideMode] = useState(() => {
		return localStorage.getItem("docs-wide-mode") === "true";
	});
	// Inline metadata editing (Notion-like)
	const [metaTitle, setMetaTitle] = useState("");
	const [metaDescription, setMetaDescription] = useState("");
	const [metaTags, setMetaTags] = useState("");

	const markdownPreviewRef = useRef<HTMLDivElement>(null);
	const scrollContainerRef = useRef<HTMLDivElement>(null);
	const scrollPositions = useRef<Map<string, number>>(new Map());
	const scrollAnimationRef = useRef<number | null>(null);

	const updateSectionHash = useCallback((headingId: string | null) => {
		const hash = headingId ? `#${encodeURIComponent(headingId)}` : "";
		const url = `${window.location.pathname}${window.location.search}${hash}`;
		window.history.replaceState(window.history.state, "", url);
	}, []);

	const scrollToHeading = useCallback((headingId: string, behavior: ScrollBehavior = "smooth") => {
		const container = scrollContainerRef.current;
		if (!container) return false;

		const viewport =
			container.querySelector<HTMLElement>("[data-radix-scroll-area-viewport]") ||
			container;
		const heading = viewport.querySelector<HTMLElement>(`#${CSS.escape(headingId)}`);
		if (!heading) return false;

		const viewportRect = viewport.getBoundingClientRect();
		const headingRect = heading.getBoundingClientRect();
		const targetTop = viewport.scrollTop + (headingRect.top - viewportRect.top) - 20;

		if (scrollAnimationRef.current !== null) {
			window.cancelAnimationFrame(scrollAnimationRef.current);
			scrollAnimationRef.current = null;
		}

		if (behavior === "auto") {
			viewport.scrollTop = targetTop;
			return true;
		}

		const startTop = viewport.scrollTop;
		const distance = targetTop - startTop;
		const duration = 280;
		const startTime = performance.now();
		const easeInOutCubic = (t: number) =>
			t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2;

		const animate = (now: number) => {
			const elapsed = now - startTime;
			const progress = Math.min(elapsed / duration, 1);
			viewport.scrollTop = startTop + distance * easeInOutCubic(progress);

			if (progress < 1) {
				scrollAnimationRef.current = window.requestAnimationFrame(animate);
			} else {
				scrollAnimationRef.current = null;
			}
		};

		scrollAnimationRef.current = window.requestAnimationFrame(animate);
		return true;
	}, []);

	const navigateToHeading = useCallback((headingId: string, behavior: ScrollBehavior = "smooth") => {
		if (scrollToHeading(headingId, behavior)) {
			updateSectionHash(headingId);
		}
	}, [scrollToHeading, updateSectionHash]);

	// Persist wide mode
	useEffect(() => {
		localStorage.setItem("docs-wide-mode", String(wideMode));
	}, [wideMode]);

	// Sync metadata state when selected doc changes
	useEffect(() => {
		if (selectedDoc) {
			setMetaTitle(selectedDoc.metadata.title || "");
			setMetaDescription(selectedDoc.metadata.description || "");
			setMetaTags(selectedDoc.metadata.tags?.join(", ") || "");
		}
	}, [selectedDoc?.path]);

	// Save metadata on blur (Notion-like auto-save)
	const handleSaveMetadata = async (field: "title" | "description" | "tags") => {
		if (!selectedDoc || selectedDoc.isImported) return;

		const updates: { title?: string; description?: string; tags?: string[] } = {};

		if (field === "title" && metaTitle !== (selectedDoc.metadata.title || "")) {
			updates.title = metaTitle;
		} else if (field === "description" && metaDescription !== (selectedDoc.metadata.description || "")) {
			updates.description = metaDescription;
		} else if (field === "tags") {
			const newTags = metaTags.split(",").map(t => t.trim()).filter(t => t);
			const oldTags = selectedDoc.metadata.tags || [];
			if (JSON.stringify(newTags) !== JSON.stringify(oldTags)) {
				updates.tags = newTags;
			}
		}

		if (Object.keys(updates).length === 0) return;

		try {
			await updateDoc(normalizePathForAPI(selectedDoc.path), updates);
			loadDocs();
		} catch (err) {
			console.error("Failed to save metadata:", err);
			// Reset on error
			setMetaTitle(selectedDoc.metadata.title || "");
			setMetaDescription(selectedDoc.metadata.description || "");
			setMetaTags(selectedDoc.metadata.tags?.join(", ") || "");
		}
	};

	// Check URL for ?create=true param
	useEffect(() => {
		if (location.search.create === true || location.search.create === "true") {
			setShowCreateView(true);
			navigateTo("/docs", { replace: true });
		}
	}, [location.search.create]);

	// Save scroll position before changing doc
	const saveScrollPosition = useCallback(() => {
		if (selectedDoc && scrollContainerRef.current) {
			scrollPositions.current.set(selectedDoc.path, scrollContainerRef.current.scrollTop);
		}
	}, [selectedDoc]);

	// Restore scroll position when doc changes
	useEffect(() => {
		if (selectedDoc && scrollContainerRef.current) {
			const activeHash = decodeURIComponent(window.location.hash.replace(/^#/, ""));
			if (activeHash) return;
			const savedPosition =
				scrollPositions.current.get(selectedDoc.path) || 0;
			requestAnimationFrame(() => {
				if (scrollContainerRef.current) {
					scrollContainerRef.current.scrollTop = savedPosition;
				}
			});
		}
	}, [selectedDoc?.path]);

	useEffect(() => {
		if (!selectedDoc) return;

		const applyHashNavigation = () => {
			const headingId = decodeURIComponent(window.location.hash.replace(/^#/, ""));
			if (!headingId) return;
			window.setTimeout(() => {
				scrollToHeading(headingId, "auto");
			}, 80);
		};

		applyHashNavigation();
		window.addEventListener("hashchange", applyHashNavigation);
		return () => window.removeEventListener("hashchange", applyHashNavigation);
	}, [scrollToHeading, selectedDoc?.path]);

	useEffect(() => {
		return () => {
			if (scrollAnimationRef.current !== null) {
				window.cancelAnimationFrame(scrollAnimationRef.current);
			}
		};
	}, []);

	// Handle markdown link clicks for internal navigation
	useEffect(() => {
		const handleLinkClick = (e: MouseEvent) => {
			let target = e.target as HTMLElement;
			while (
				target &&
				target.tagName !== "A" &&
				target !== markdownPreviewRef.current
			) {
				target = target.parentElement as HTMLElement;
			}

			if (target && target.tagName === "A") {
				const anchor = target as HTMLAnchorElement;
				const href = anchor.getAttribute("href");

				if (href && href.startsWith("#")) {
					e.preventDefault();
					navigateToHeading(decodeURIComponent(href.slice(1)));
					return;
				}

				if (href && /^@?task-[\w.]+(.md)?$/.test(href)) {
					e.preventDefault();
					const taskId = href
						.replace(/^@/, "")
						.replace(/^task-/, "")
						.replace(".md", "");
					openTask(taskId);
					return;
				}

				if (href && href.startsWith("@doc/")) {
					e.preventDefault();
					const docPath = href.replace("@doc/", "");
					navigateTo(`/docs/${docPath}.md`);
					return;
				}

				if (href && (href.endsWith(".md") || href.includes(".md#"))) {
					e.preventDefault();
					const normalizedHref = href.replace(/^\.\//, "").replace(/^\//, "");
					const [docPathPart, hashPart] = normalizedHref.split("#");
					const docPath = docPathPart ?? normalizedHref;
					void navigateTo(`/docs/${docPath}`).then(() => {
						if (hashPart) {
							window.setTimeout(() => {
								navigateToHeading(decodeURIComponent(hashPart), "auto");
							}, 80);
						}
					});
				}
			}
		};

		const previewEl = markdownPreviewRef.current;
		if (previewEl) {
			previewEl.addEventListener("click", handleLinkClick);
			return () => previewEl.removeEventListener("click", handleLinkClick);
		}
	}, [docs, navigateToHeading, openTask, selectedDoc]);

	const handleCreateDoc = async () => {
		if (!newDocTitle.trim()) return;

		setCreating(true);
		try {
			const tags = newDocTags
				.split(",")
				.map((t) => t.trim())
				.filter((t) => t);

			await createDoc({
				title: newDocTitle,
				description: newDocDescription,
				tags,
				folder: newDocFolder,
				content: newDocContent,
			});

			setNewDocTitle("");
			setNewDocDescription("");
			setNewDocTags("");
			setNewDocFolder("");
			setNewDocContent("");
			setShowCreateView(false);
			setMobileSidebarOpen(false);
			loadDocs();
		} catch (err) {
			console.error("Failed to create doc:", err);
		} finally {
			setCreating(false);
		}
	};

	const handleEdit = () => {
		if (selectedDoc) {
			setEditedContent(selectedDoc.content);
			setIsEditing(true);
		}
	};

	const handleCopyPath = () => {
		if (selectedDoc) {
			const normalizedPath = toDisplayPath(selectedDoc.path).replace(
				/\.md$/,
				"",
			);
			const refPath = `@doc/${normalizedPath}`;
			navigator.clipboard.writeText(refPath).then(() => {
				setPathCopied(true);
				setTimeout(() => setPathCopied(false), 2000);
			});
		}
	};

	const handleSave = async () => {
		if (!selectedDoc) return;

		setSaving(true);
		try {
			await updateDoc(normalizePathForAPI(selectedDoc.path), {
				content: editedContent,
			});
			loadDocs();
			setIsEditing(false);
		} catch (err) {
			console.error("Failed to save doc:", err);
		} finally {
			setSaving(false);
		}
	};

	const handleCancel = () => {
		setIsEditing(false);
		setEditedContent("");
	};

	const openCreateView = () => {
		setNewDocFolder(currentFolder || "");
		setShowCreateView(true);
		setMobileSidebarOpen(false);
	};

	const sidebarContent = (
		<DocsFileManager
			onCreateDoc={openCreateView}
			docs={docs}
			currentFolder={currentFolder}
			navigateToFolder={navigateToFolder}
			setSelectedDoc={setSelectedDoc}
			showSpecsOnly={showSpecsOnly}
			setShowSpecsOnly={setShowSpecsOnly}
			searchQuery={docSearchQuery}
			onSearchQueryChange={setDocSearchQuery}
			selectedDocPath={selectedDoc?.path}
			onItemSelect={() => setMobileSidebarOpen(false)}
			className="h-full"
		/>
	);

	if (loading) {
		return (
			<div className="p-6 flex items-center justify-center h-64">
				<div className="text-lg text-muted-foreground">
					Loading documentation...
				</div>
			</div>
		);
	}

	if (error) {
		return (
			<div className="p-6 flex items-center justify-center h-64">
				<div className="text-center">
					<p className="text-lg text-destructive mb-2">
						Failed to load documentation
					</p>
					<p className="text-sm text-muted-foreground mb-4">{error}</p>
					<Button
						onClick={() => loadDocs()}
						variant="outline"
					>
						Retry
					</Button>
				</div>
			</div>
		);
	}

	return (
		<div className="h-full flex overflow-hidden bg-background">
			<aside className="hidden lg:flex w-[300px] xl:w-[320px] shrink-0 bg-[#fafaf8] dark:bg-muted/10 border-r border-border/40">
				<div className="h-full w-full px-0 py-5">{sidebarContent}</div>
			</aside>

			<div className="min-w-0 flex-1 flex flex-col overflow-hidden">
				<Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
					<SheetContent side="left" className="w-[92vw] max-w-none p-0 sm:max-w-md">
						<div className="flex h-full flex-col">
							<div className="border-b border-border/50 px-4 py-3">
								<SheetTitle>Browse docs</SheetTitle>
							</div>
							<div className="min-h-0 flex-1 px-4 py-4">{sidebarContent}</div>
						</div>
					</SheetContent>
				</Sheet>

				{selectedDoc ? (
					<>
						<div className="flex items-center gap-1.5 sm:gap-2 px-3 sm:px-5 py-2 border-b border-border/40 shrink-0 bg-background/90 backdrop-blur-sm">
							<Button
								variant="ghost"
								size="sm"
								onClick={() => setMobileSidebarOpen(true)}
								className="h-7 px-2 text-muted-foreground hover:text-foreground lg:hidden"
							>
								<Menu className="w-3.5 h-3.5" />
							</Button>
							<Button
								variant="ghost"
								size="sm"
								onClick={() => navigateToFolder(currentFolder)}
								className="h-8 px-2 text-muted-foreground hover:text-foreground"
							>
								<ArrowLeft className="w-3.5 h-3.5 sm:mr-1" />
								<span className="hidden sm:inline text-xs">Back</span>
							</Button>
							<button
								type="button"
								onClick={handleCopyPath}
								className="flex items-center gap-1.5 text-[11px] text-muted-foreground hover:text-foreground transition-colors min-w-0 rounded-full px-2 py-1 hover:bg-accent/60"
								title="Click to copy reference"
							>
								<Copy className="w-3 h-3 shrink-0" />
								<span className="font-mono truncate max-w-[180px] sm:max-w-[240px] opacity-85">
									@doc/{toDisplayPath(selectedDoc.path).replace(/\.md$/, "")}
								</span>
							</button>
							{pathCopied && <span className="text-green-600 text-[11px]">Copied</span>}
							<div className="flex-1" />
							{!isEditing && (
								<Button
									variant="ghost"
									size="sm"
									onClick={() => setWideMode(!wideMode)}
									className="h-8 px-2 text-muted-foreground hover:text-foreground"
									title={wideMode ? "Normal width" : "Full width"}
								>
									{wideMode ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
								</Button>
							)}
							{!isEditing ? (
								<Button
									size="sm"
									variant="ghost"
									onClick={handleEdit}
									disabled={selectedDoc.isImported}
									className="h-8 px-2"
									title={selectedDoc.isImported ? "Imported docs are read-only" : "Edit document"}
								>
									<Pencil className="w-3.5 h-3.5 sm:mr-1" />
									<span className="hidden sm:inline text-xs">Edit</span>
								</Button>
							) : (
								<>
									<Button size="sm" onClick={handleSave} disabled={saving} className="h-8 px-2.5 rounded-full">
										<Check className="w-3.5 h-3.5 sm:mr-1" />
										<span className="hidden sm:inline text-xs">{saving ? "Saving..." : "Save"}</span>
									</Button>
									<Button size="sm" variant="secondary" onClick={handleCancel} disabled={saving} className="h-8 px-2.5 rounded-full">
										<X className="w-3.5 h-3.5 sm:mr-1" />
										<span className="hidden sm:inline text-xs">Cancel</span>
									</Button>
								</>
							)}
						</div>

						{isEditing ? (
							<div className="flex-1 min-h-0 overflow-hidden p-4 sm:p-6">
								<MDEditor
									markdown={editedContent}
									onChange={setEditedContent}
									placeholder="Write your documentation here..."
									height="100%"
									className="h-full"
								/>
							</div>
						) : (
							<div className="flex-1 overflow-y-auto" ref={scrollContainerRef}>
								<div className="flex justify-center">
									<article className={`w-full px-6 sm:px-8 py-10 sm:py-12 transition-[max-width] duration-300 ease-in-out ${wideMode ? "max-w-[1040px]" : "max-w-[760px]"}`}>
										{/* Document header — Notion-like inline editing */}
										<header className="mb-10">
									{/* Title — inline editable */}
									{selectedDoc.isImported ? (
												<h1 className="text-4xl font-semibold tracking-tight mb-2 text-balance">
											{selectedDoc.metadata.title}
										</h1>
									) : (
										<input
											type="text"
											value={metaTitle}
											onChange={(e) => setMetaTitle(e.target.value)}
											onBlur={() => handleSaveMetadata("title")}
											onKeyDown={(e) => e.key === "Enter" && e.currentTarget.blur()}
													className="text-4xl font-semibold tracking-tight bg-transparent w-full outline-none border-none p-0 mb-2 placeholder:text-muted-foreground/35"
											placeholder="Untitled"
										/>
									)}

									{/* Description — inline editable */}
											{selectedDoc.isImported ? (
												selectedDoc.metadata.description && (
													<p className="text-[15px] leading-7 text-muted-foreground mb-4 max-w-2xl">
														{selectedDoc.metadata.description}
													</p>
										)
									) : (
										<input
											type="text"
											value={metaDescription}
											onChange={(e) => setMetaDescription(e.target.value)}
											onBlur={() => handleSaveMetadata("description")}
											onKeyDown={(e) => e.key === "Enter" && e.currentTarget.blur()}
													className="text-[15px] leading-7 text-muted-foreground bg-transparent w-full outline-none border-none p-0 mb-4 placeholder:text-muted-foreground/35 max-w-2xl"
													placeholder="Add a description..."
												/>
											)}

											<div className="mb-4 space-y-2">
												{!selectedDoc.isImported ? (
													<input
														type="text"
														value={metaTags}
														onChange={(e) => setMetaTags(e.target.value)}
														onBlur={() => handleSaveMetadata("tags")}
														onKeyDown={(e) => e.key === "Enter" && e.currentTarget.blur()}
														className="bg-transparent outline-none border-none p-0 text-[12px] text-muted-foreground/75 placeholder:text-muted-foreground/40 w-full"
														placeholder="Add tags..."
													/>
												) : (
													selectedDoc.metadata.tags &&
													selectedDoc.metadata.tags.length > 0 && (
														<div className="flex flex-wrap items-center gap-1.5">
															{selectedDoc.metadata.tags.map((tag) => (
																<span
																	key={tag}
																	className="rounded-full bg-muted/50 px-2 py-0.5 text-[10px] text-muted-foreground"
																>
																	{tag}
																</span>
															))}
														</div>
													)
												)}
												<div className="text-[11px] font-mono text-muted-foreground/65 break-all">
													@doc/{toDisplayPath(selectedDoc.path).replace(/\.md$/, "")}
												</div>
											</div>

											{/* Metadata row */}
											<div className="flex items-center gap-2.5 flex-wrap text-[11px] text-muted-foreground/85">
											{isSpec(selectedDoc) && (
												<span className="px-2 py-0.5 text-[10px] font-medium bg-sky-100 text-sky-800 dark:bg-sky-950/60 dark:text-sky-200 rounded-full">
													SPEC
												</span>
											)}
										{isSpec(selectedDoc) && getSpecStatus(selectedDoc) && (
											<span
													className={`px-2 py-0.5 text-[10px] font-medium rounded-full ${
													getSpecStatus(selectedDoc) === "approved"
														? "bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300"
														: getSpecStatus(selectedDoc) === "implemented"
															? "bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300"
															: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300"
												}`}
											>
												{(getSpecStatus(selectedDoc) ?? "").charAt(0).toUpperCase() +
													(getSpecStatus(selectedDoc) ?? "").slice(1)}
											</span>
										)}
										{selectedDoc.isImported && (
												<span className="px-2 py-0.5 text-[10px] font-medium bg-muted text-muted-foreground rounded-full">
												Imported
											</span>
										)}

											<span>
												Updated {new Date(selectedDoc.metadata.updatedAt).toLocaleDateString()}
											</span>
									</div>
									{/* Spec AC Progress */}
									{isSpec(selectedDoc) &&
										(() => {
											const acProgress = parseACProgress(selectedDoc.content);
											return acProgress.total > 0 ? (
												<div className="flex items-center gap-2 mt-4 rounded-2xl bg-muted/35 px-3 py-2 w-fit">
													<ListChecks className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
													<Progress
														value={Math.round(
															(acProgress.completed / acProgress.total) * 100,
														)}
														className="flex-1 h-1.5 max-w-[180px]"
													/>
													<span className="text-xs text-muted-foreground">
														{acProgress.completed}/{acProgress.total}
													</span>
												</div>
											) : null;
										})()}
									{/* Linked tasks */}
									{isSpec(selectedDoc) && (
											<div className="mt-4 rounded-2xl bg-muted/25 px-3 py-2.5">
											<button
												type="button"
												onClick={() => setLinkedTasksExpanded(!linkedTasksExpanded)}
													className="flex items-center gap-1.5 text-[11px] text-muted-foreground hover:text-foreground transition-colors"
											>
												<FileText className="w-3.5 h-3.5" />
												<span>{linkedTasks.length} linked tasks</span>
												{linkedTasksExpanded ? (
													<ChevronUp className="w-3 h-3" />
												) : (
													<ChevronDown className="w-3 h-3" />
												)}
											</button>
											{linkedTasksExpanded && (
													<div className="space-y-1 mt-2">
														{linkedTasks.length === 0 ? (
															<p className="text-xs text-muted-foreground">
																No tasks are linked to this spec yet.
															</p>
														) : linkedTasks.map((task) => (
															<button
																type="button"
																key={task.id}
															onClick={() => openTask(task.id)}
															className="flex items-center gap-1.5 p-1.5 rounded-xl hover:bg-accent/60 transition-colors w-full text-left"
														>
															<span
																className={`w-1.5 h-1.5 rounded-full shrink-0 ${
																	task.status === "done"
																		? "bg-green-500"
																		: task.status === "in-progress"
																			? "bg-yellow-500"
																			: task.status === "blocked"
																				? "bg-red-500"
																				: "bg-gray-400"
																}`}
															/>
															<span className="text-xs truncate">
																{task.title}
															</span>
														</button>
														))}
													</div>
											)}
										</div>
									)}
								</header>

										{/* Document content */}
										<div ref={markdownPreviewRef} className="prose-neutral dark:prose-invert">
											<MDRender
												markdown={selectedDoc.content || ""}
												onTaskLinkClick={setPreviewTaskId}
												onDocLinkClick={(path) => {
													// In DocsPage, navigate directly without preview
													navigateTo(`/docs/${path}`);
												}}
												onHeadingAnchorClick={navigateToHeading}
												showHeadingAnchors
											/>
										</div>
									</article>

									{/* Right TOC */}
									{!isEditing && (
										<div className="w-52 shrink-0 hidden xl:block pt-12 pr-6">
										<div className="sticky top-8">
												<DocsTOC
													markdown={selectedDoc.content || ""}
													scrollContainerRef={scrollContainerRef}
													onHeadingSelect={navigateToHeading}
												/>
											</div>
										</div>
									)}
								</div>
							</div>
						)}
					</>
				) : showCreateView ? (
					<>
							<div className="flex items-center gap-1.5 sm:gap-2 px-3 sm:px-5 py-2 border-b border-border/40 shrink-0 bg-background/90 backdrop-blur-sm">
							<Button
								variant="ghost"
								size="sm"
								onClick={() => setMobileSidebarOpen(true)}
								className="h-7 px-2 text-muted-foreground hover:text-foreground lg:hidden"
							>
								<Menu className="w-3.5 h-3.5" />
							</Button>
							<Button
								variant="ghost"
								size="sm"
								onClick={() => {
									setShowCreateView(false);
									setNewDocTitle("");
									setNewDocDescription("");
									setNewDocTags("");
									setNewDocFolder("");
									setNewDocContent("");
								}}
								disabled={creating}
								className="h-7 px-2 text-muted-foreground hover:text-foreground"
							>
								<ArrowLeft className="w-3.5 h-3.5 sm:mr-1" />
								<span className="hidden sm:inline text-xs">Back</span>
							</Button>
							<div className="flex-1" />
							<Button size="sm" onClick={handleCreateDoc} disabled={creating || !newDocTitle.trim()} className="h-7 px-3">
								<Check className="w-3.5 h-3.5 sm:mr-1" />
								<span className="text-xs">{creating ? "Creating..." : "Create"}</span>
							</Button>
						</div>

						<div className="flex-1 flex flex-col min-h-0 overflow-y-auto">
							<div className="px-6 pt-6 pb-4 shrink-0 flex justify-center">
								<div className="w-full max-w-[720px]">
									<input
										type="text"
								value={newDocTitle}
								onChange={(e) => setNewDocTitle(e.target.value)}
								className="text-3xl font-semibold tracking-tight bg-transparent w-full outline-none border-none p-0 mb-1 placeholder:text-muted-foreground/40"
								placeholder="Untitled"
								autoFocus
							/>
							<input
								type="text"
								value={newDocDescription}
								onChange={(e) => setNewDocDescription(e.target.value)}
								className="text-base text-muted-foreground bg-transparent w-full outline-none border-none p-0 mb-4 placeholder:text-muted-foreground/40"
								placeholder="Add a description..."
							/>
							<div className="flex items-center gap-3 text-xs text-muted-foreground">
								<div className="flex items-center gap-1.5">
									<span className="text-muted-foreground/60">Folder</span>
									<input
										type="text"
										value={newDocFolder}
										onChange={(e) => setNewDocFolder(e.target.value)}
										className="bg-transparent outline-none border-none p-0 text-xs text-foreground placeholder:text-muted-foreground/40 w-[120px]"
										placeholder="root"
									/>
								</div>
								<span className="text-border">|</span>
								<div className="flex items-center gap-1.5 flex-1">
									<span className="text-muted-foreground/60 shrink-0">Tags</span>
									<input
										type="text"
										value={newDocTags}
										onChange={(e) => setNewDocTags(e.target.value)}
										className="bg-transparent outline-none border-none p-0 text-xs text-foreground placeholder:text-muted-foreground/40 flex-1"
										placeholder="guide, tutorial, api"
									/>
								</div>
									</div>
								</div>
							</div>
							<div className="flex-1 min-h-0 px-6 pb-6">
								<MDEditor
							markdown={newDocContent}
							onChange={setNewDocContent}
							placeholder="Write your documentation here..."
							height="100%"
									className="h-full"
								/>
							</div>
						</div>
					</>
			) : (
						<div className="flex-1 min-h-0 flex items-center justify-center p-6 sm:p-10">
							<div className="max-w-md text-center">
								<div className="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-[20px] bg-muted/50 text-muted-foreground">
									<FileText className="w-6 h-6" />
								</div>
								<h2 className="text-2xl font-semibold tracking-tight">Browse your docs</h2>
								<p className="mt-3 text-sm leading-6 text-muted-foreground">
							Pick a document from the left sidebar or create a new one in
							 <span className="font-medium text-foreground">{currentFolder || "root"}</span>.
						</p>
						<div className="mt-4 flex items-center justify-center gap-2">
							<Button onClick={openCreateView}>
								<Plus className="w-4 h-4 mr-1.5" />
								New Doc
							</Button>
							<Button variant="outline" onClick={() => setMobileSidebarOpen(true)} className="lg:hidden">
								<Menu className="w-4 h-4 mr-1.5" />
								Browse
							</Button>
						</div>
					</div>
				</div>
			)}
			</div>
			<TaskPreviewDialog
				taskId={previewTaskId}
				open={!!previewTaskId}
				onOpenChange={(open) => {
					if (!open) setPreviewTaskId(null);
				}}
			/>
		</div>
	);
}
