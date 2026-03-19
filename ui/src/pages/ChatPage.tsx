import { useEffect, useMemo, useRef } from "react";
import { Bot, Loader2, Plus } from "lucide-react";

import { Sheet, SheetContent } from "../components/ui/sheet";
import {
	ChatHeader,
	ChatInput,
	ChatMessages,
	ChatSidebar,
	OpenCodeUnavailableBanner,
} from "../components/organisms/ChatPage";
import { TaskPreviewDialog } from "../components/organisms/TaskDetail/TaskPreviewDialog";
import { DocPreviewDialog } from "../components/organisms/DocsPreview/DocPreviewDialog";
import { useChatPage } from "./chat/useChatPage";
import { SubSessionsContext } from "../contexts/SubSessionsContext";

export default function ChatPage() {
	const previousActiveIdRef = useRef<string | null>(null);
	const {
		loading,
		messagesLoading,
		mobileSidebarOpen,
		setMobileSidebarOpen,
		localSessions,
		activeId,
		activeSession,
		activeSessionTodos,
		activeQuestion,
		activePermission,
		activeQuickCommands,
		rootSessions,
		sessionActivity,
		queueCount,
		chatDisabled,
		opencodeBlockedReason,
		pickerProviders,
		catalog,
		modelSettings,
		autoModelLabel,
		lastLoadedAt,
		slashItems,
		previewTaskId,
		setPreviewTaskId,
		previewDocPath,
		setPreviewDocPath,
		setDefaultModel,
		setDefaultVariant,
		updateModelPref,
		toggleProviderHidden,
		inputRestoreValue,
		setInputRestoreValue,
		handleSelectSession,
		createNewChat,
		handleDelete,
		handleRename,
		handleModelChange,
		handleSend,
		handleSubmitQuestion,
		handleRejectQuestion,
		handleRespondPermission,
		handleRevertMessage,
		handleStop,
	} = useChatPage();

	useEffect(() => {
		const previousActiveId = previousActiveIdRef.current;
		if (mobileSidebarOpen && previousActiveId !== activeId) {
			setMobileSidebarOpen(false);
		}
		previousActiveIdRef.current = activeId;
	}, [activeId, mobileSidebarOpen, setMobileSidebarOpen]);

	if (loading) {
		return (
			<div className="flex flex-1 items-center justify-center">
				<Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
			</div>
		);
	}

	const sidebarProps = {
		sessions: rootSessions,
		activeId,
		sessionActivity,
		onNew: createNewChat,
		onDelete: handleDelete,
		onRename: handleRename,
		disabled: chatDisabled,
		actionsDisabled: chatDisabled,
	};

	const subSessionsStore = useMemo(() => ({
		sessions: localSessions,
		getById: (sessionId: string | null | undefined) => {
			if (!sessionId) return null;
			return localSessions.find((session) => session.id === sessionId) || null;
		},
		findByParent: (parentSessionId: string | undefined, referenceCreatedAt?: string) => {
			if (!parentSessionId) return null;
			const candidates = localSessions.filter((session) => session.parentSessionId === parentSessionId);
			if (candidates.length === 0) return null;

			const referenceTime = referenceCreatedAt ? new Date(referenceCreatedAt).getTime() : Number.NEGATIVE_INFINITY;
			const afterReference = candidates
				.filter((session) => new Date(session.createdAt).getTime() >= referenceTime - 1000)
				.sort((left, right) => new Date(left.createdAt).getTime() - new Date(right.createdAt).getTime());

			return afterReference[0]
				|| candidates.sort((left, right) => new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime())[0]
				|| null;
		},
	}), [localSessions]);

	return (
		<SubSessionsContext.Provider value={subSessionsStore}>
		<div className="flex h-full bg-background">
			{/* Desktop sidebar */}
			<div className="hidden md:flex">
				<ChatSidebar {...sidebarProps} onSelect={handleSelectSession} />
			</div>

			{/* Mobile sidebar sheet */}
			<Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
				<SheetContent side="left" className="p-0 w-[296px]">
					<ChatSidebar
						{...sidebarProps}
						onNew={() => {
							createNewChat();
							setMobileSidebarOpen(false);
						}}
						onSelect={(id) => {
							handleSelectSession(id);
							setMobileSidebarOpen(false);
						}}
					/>
				</SheetContent>
			</Sheet>

			<div className="flex min-w-0 flex-1">
				{activeSession ? (
					<div className="flex min-w-0 flex-1 flex-col">
						<ChatHeader session={activeSession} onMenuToggle={() => setMobileSidebarOpen(true)} />
						{opencodeBlockedReason && <OpenCodeUnavailableBanner message={opencodeBlockedReason} />}
						<ChatMessages
							session={activeSession}
							loading={messagesLoading}
							onSend={handleSend}
							onSubmitQuestion={handleSubmitQuestion}
							onRejectQuestion={handleRejectQuestion}
							onRevert={handleRevertMessage}
							onPreviewTask={setPreviewTaskId}
							onPreviewDoc={setPreviewDocPath}
						/>
							<ChatInput
								onSend={handleSend}
								onSubmitQuestion={handleSubmitQuestion}
								onRejectQuestion={handleRejectQuestion}
								onRespondPermission={handleRespondPermission}
								onStop={handleStop}
							isStreaming={activeSession.status === "streaming"}
							disabled={chatDisabled}
							queueCount={queueCount[activeId ?? ""] || 0}
							providers={pickerProviders}
							catalog={catalog}
							currentModel={activeSession.model?.key || null}
							currentVariant={activeSession.model?.key ? modelSettings.variantModels?.[activeSession.model.key] || null : null}
							onModelChange={handleModelChange}
							onSetDefaultModel={setDefaultModel}
							onSetDefaultVariant={setDefaultVariant}
							onUpdateModelPref={updateModelPref}
							onToggleProviderHidden={toggleProviderHidden}
							lastLoadedAt={lastLoadedAt}
							slashItems={slashItems}
							autoModelLabel={autoModelLabel}
							todos={activeSessionTodos}
							quickCommands={activeQuickCommands}
							activeQuestion={activeQuestion}
							activePermission={activePermission}
							restoreValue={inputRestoreValue}
							onRestoreValueConsumed={() => setInputRestoreValue(null)}
						/>
					</div>
				) : (
					<div className="flex flex-1 flex-col">
						<div className="flex shrink-0 items-center border-b border-border/50 px-4 py-3 md:hidden">
							<button
								type="button"
								onClick={() => setMobileSidebarOpen(true)}
								className="flex items-center justify-center rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
								title="Open sidebar"
							>
								<Bot className="h-5 w-5" />
							</button>
						</div>
						<div className="flex flex-1 flex-col items-center justify-center gap-4 px-6">
							{opencodeBlockedReason && <OpenCodeUnavailableBanner message={opencodeBlockedReason} />}
							<div className="flex h-20 w-20 items-center justify-center rounded-3xl border border-border/60 bg-background/80 shadow-sm">
								<Bot className="h-10 w-10 text-muted-foreground/30" />
							</div>
							<div className="space-y-1 text-center">
								<p className="text-base font-medium text-foreground">Pick a conversation</p>
								<p className="text-sm text-muted-foreground">Choose an existing chat or start a fresh one.</p>
							</div>
							<button
								type="button"
								onClick={createNewChat}
								disabled={chatDisabled}
								className="flex items-center gap-2 rounded-full border border-border bg-background px-4 py-2 text-sm text-foreground shadow-sm transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50"
							>
								<Plus className="h-4 w-4" />
								New Chat
							</button>
						</div>
					</div>
				)}
			</div>

			<TaskPreviewDialog
				taskId={previewTaskId}
				open={!!previewTaskId}
				onOpenChange={(open) => { if (!open) setPreviewTaskId(null); }}
			/>
			<DocPreviewDialog
				docPath={previewDocPath}
				open={!!previewDocPath}
				onOpenChange={(open) => { if (!open) setPreviewDocPath(null); }}
			/>
		</div>
		</SubSessionsContext.Provider>
	);
}
