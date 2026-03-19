import { ArrowLeft, Check, Copy, ExternalLink, Loader2, Plus, Search, Trash2, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import type { ProviderAuthAuthorization, ProviderAuthMethod } from "../../../api/client";
import { useOpenCode } from "../../../contexts/OpenCodeContext";
import { cn } from "../../../lib/utils";
import { Badge } from "../../ui/badge";
import { Button } from "../../ui/button";
import { Dialog, DialogContent } from "../../ui/dialog";
import { Input } from "../../ui/input";
import { toast } from "../../ui/sonner";

const POPULAR = new Set([
	"opencode-zen",
	"opencode-go",
	"anthropic",
	"github-copilot",
	"openai",
	"google",
	"openrouter",
	"vercel",
	"xai",
	"mistral",
]);

const DESCRIPTIONS: Record<string, string> = {
	"opencode-zen": "Reliable optimized models",
	"opencode-go": "Low cost subscription for everyone",
	"anthropic": "Direct access to Claude models, including Pro and Max",
	"github-copilot": "AI models for coding assistance via GitHub Copilot",
	"openai": "GPT models for fast, capable general AI tasks",
};

const BADGES: Record<string, string> = {
	"opencode-zen": "Recommended",
	"opencode-go": "Recommended",
};

interface ProviderConnectDialogProps {
	open: boolean;
	onClose: () => void;
}

interface ProviderSummary {
	id: string;
	name: string;
}

interface ActiveProviderState extends ProviderSummary {
	selectedMethodIndex: number | null;
}

interface OAuthState extends ProviderSummary {
	methodIndex: number;
	authorization: ProviderAuthAuthorization | null;
}

interface ProviderMethodEntry extends ProviderAuthMethod {
	methodIndex: number;
}

function getProviderMethodEntries(methods: ProviderAuthMethod[]): ProviderMethodEntry[] {
	return methods.map((method, methodIndex) => ({ ...method, methodIndex }));
}

function getErrorMessage(error: unknown, fallback: string) {
	return error instanceof Error && error.message ? error.message : fallback;
}

function getMethodButtonLabel(method?: ProviderMethodEntry | null) {
	if (!method) return "Continue";
	return method.type === "api" ? "Connect" : "Continue";
}

function getAutoInstructionText(authorization: ProviderAuthAuthorization) {
	return authorization.instructions || "Complete authorization in your browser. This window will close automatically.";
}

export function ProviderConnectDialog({ open, onClose }: ProviderConnectDialogProps) {
	const { providerResponse, providerAuthMethods, connectWithApiKey, startOAuth, finishOAuth, disconnectProvider } = useOpenCode();
	const [query, setQuery] = useState("");
	const [apiKeys, setApiKeys] = useState<Record<string, string>>({});
	const [savingProviderId, setSavingProviderId] = useState<string | null>(null);
	const [disconnectingProviderId, setDisconnectingProviderId] = useState<string | null>(null);
	const [authorizingState, setAuthorizingState] = useState<{ providerId: string; methodIndex: number } | null>(null);
	const [providerErrors, setProviderErrors] = useState<Record<string, string>>({});
	const [activeProvider, setActiveProvider] = useState<ActiveProviderState | null>(null);
	const [oauthState, setOauthState] = useState<OAuthState | null>(null);

	const connectedIds = useMemo(() => {
		if (!providerResponse) return new Set<string>();
		if (Array.isArray(providerResponse.connected)) {
			return new Set(providerResponse.connected);
		}
		return new Set((providerResponse.all ?? []).map((provider) => provider.id));
	}, [providerResponse]);

	const filtered = useMemo(() => {
		const normalized = query.trim().toLowerCase();
		const all = providerResponse?.all ?? [];
		if (!normalized) return all;
		return all.filter(
			(provider) =>
				provider.name.toLowerCase().includes(normalized) ||
				provider.id.toLowerCase().includes(normalized) ||
				DESCRIPTIONS[provider.id]?.toLowerCase().includes(normalized),
		);
	}, [providerResponse, query]);

	const popular = filtered.filter((provider) => POPULAR.has(provider.id));
	const others = filtered.filter((provider) => !POPULAR.has(provider.id));
	const activeMethods = useMemo(
		() => (activeProvider ? getProviderMethodEntries(providerAuthMethods?.[activeProvider.id] ?? []) : []),
		[activeProvider, providerAuthMethods],
	);
	const selectedMethod = useMemo(
		() => activeMethods.find((method) => method.methodIndex === activeProvider?.selectedMethodIndex) ?? null,
		[activeMethods, activeProvider],
	);
	const activeProviderError = activeProvider ? providerErrors[activeProvider.id] : undefined;

	const clearProviderError = (providerId: string) => {
		setProviderErrors((prev) => {
			if (!prev[providerId]) return prev;
			const next = { ...prev };
			delete next[providerId];
			return next;
		});
	};

	const setProviderError = (providerId: string, message: string) => {
		setProviderErrors((prev) => ({ ...prev, [providerId]: message }));
	};

	const resetDialogState = () => {
		setQuery("");
		setApiKeys({});
		setSavingProviderId(null);
		setDisconnectingProviderId(null);
		setAuthorizingState(null);
		setProviderErrors({});
		setActiveProvider(null);
		setOauthState(null);
	};

	const handleClose = () => {
		resetDialogState();
		onClose();
	};

	const handleOpenProvider = (provider: ProviderSummary) => {
		const methods = getProviderMethodEntries(providerAuthMethods?.[provider.id] ?? []);
		clearProviderError(provider.id);
		if (methods.length === 0) {
			setProviderError(provider.id, `${provider.name} does not expose a supported authentication method yet.`);
			return;
		}
		setActiveProvider({
			id: provider.id,
			name: provider.name,
			selectedMethodIndex: methods[0]?.methodIndex ?? null,
		});
	};

	const handleStartOAuth = async (provider: ProviderSummary, methodIndex: number) => {
		clearProviderError(provider.id);
		setAuthorizingState({ providerId: provider.id, methodIndex });
		setOauthState({ id: provider.id, name: provider.name, methodIndex, authorization: null });
		try {
			const authorization = await startOAuth(provider.id, methodIndex);
			setOauthState((prev) => (prev ? { ...prev, authorization } : null));
			if (authorization.url) {
				window.open(authorization.url, "_blank", "noopener,noreferrer");
				toast.success(`Opened ${provider.name} authorization`, {
					description: "Finish the sign-in flow in your browser, then return here.",
					position: "bottom-right",
				});
			}
		} catch (error) {
			setOauthState(null);
			setProviderError(provider.id, getErrorMessage(error, `Failed to start ${provider.name} authorization`));
		} finally {
			setAuthorizingState(null);
		}
	};

	const handleSaveKey = async (providerId: string) => {
		const apiKey = apiKeys[providerId]?.trim();
		if (!apiKey) return;
		clearProviderError(providerId);
		setSavingProviderId(providerId);
		try {
			await connectWithApiKey(providerId, apiKey);
			const providerName = providerResponse?.all?.find((provider) => provider.id === providerId)?.name ?? "Provider";
			toast.success(`${providerName} connected`, {
				description: "The provider is now ready to use in your workspace.",
				position: "bottom-right",
			});
			handleClose();
		} catch (error) {
			setProviderError(providerId, getErrorMessage(error, "Failed to connect provider"));
		} finally {
			setSavingProviderId(null);
		}
	};

	const handleContinue = async () => {
		if (!activeProvider || !selectedMethod) return;
		if (selectedMethod.type === "api") {
			await handleSaveKey(activeProvider.id);
			return;
		}
		await handleStartOAuth(activeProvider, selectedMethod.methodIndex);
	};

	const handleDisconnect = async (providerId: string) => {
		clearProviderError(providerId);
		setDisconnectingProviderId(providerId);
		try {
			await disconnectProvider(providerId);
			const providerName = providerResponse?.all?.find((provider) => provider.id === providerId)?.name ?? "Provider";
			toast.success(`${providerName} removed`, {
				description: "You can reconnect it at any time.",
				position: "bottom-right",
			});
		} catch (error) {
			setProviderError(providerId, getErrorMessage(error, "Failed to disconnect provider"));
		} finally {
			setDisconnectingProviderId(null);
		}
	};

	const handleFinishOAuth = async (code?: string) => {
		if (!oauthState) return;
		await finishOAuth(oauthState.id, oauthState.methodIndex, code);
		toast.success(`${oauthState.name} connected`, {
			description: "The provider is now ready to use in your workspace.",
			position: "bottom-right",
		});
		handleClose();
	};

	const continueDisabled =
		!selectedMethod ||
		(selectedMethod.type === "api" && !apiKeys[activeProvider?.id ?? ""]?.trim()) ||
		savingProviderId === activeProvider?.id ||
		(authorizingState?.providerId === activeProvider?.id && authorizingState.methodIndex === selectedMethod?.methodIndex);

	return (
		<Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
			<DialogContent hideCloseButton className="grid h-[78vh] max-h-[78vh] max-w-3xl grid-rows-[auto,minmax(0,1fr)] gap-0 overflow-hidden border-border/60 bg-background p-0 sm:h-[74vh]">
				{oauthState ? (
					<OAuthStepView
						providerName={oauthState.name}
						authorization={oauthState.authorization}
						onBack={() => setOauthState(null)}
						onClose={handleClose}
						onFinish={handleFinishOAuth}
					/>
				) : activeProvider ? (
					<ProviderMethodStepView
						provider={activeProvider}
						methods={activeMethods}
						selectedMethodIndex={activeProvider.selectedMethodIndex}
						apiKey={apiKeys[activeProvider.id] ?? ""}
						error={activeProviderError}
						saving={savingProviderId === activeProvider.id}
						authorizing={authorizingState?.providerId === activeProvider.id && authorizingState.methodIndex === selectedMethod?.methodIndex}
						onBack={() => setActiveProvider(null)}
						onClose={handleClose}
						onSelectMethod={(methodIndex) => {
							clearProviderError(activeProvider.id);
							setActiveProvider((prev) => (prev ? { ...prev, selectedMethodIndex: methodIndex } : prev));
						}}
						onApiKeyChange={(value) => {
							clearProviderError(activeProvider.id);
							setApiKeys((prev) => ({ ...prev, [activeProvider.id]: value }));
						}}
						onContinue={() => void handleContinue()}
						continueDisabled={continueDisabled}
					/>
				) : (
					<ProviderListStepView
						query={query}
						onQueryChange={setQuery}
						onClose={handleClose}
						popular={popular}
						others={others}
						connectedIds={connectedIds}
						disconnectingProviderId={disconnectingProviderId}
						providerErrors={providerErrors}
						onConnect={handleOpenProvider}
						onDisconnect={handleDisconnect}
					/>
				)}
			</DialogContent>
		</Dialog>
	);
}

function ProviderListStepView({
	query,
	onQueryChange,
	onClose,
	popular,
	others,
	connectedIds,
	disconnectingProviderId,
	providerErrors,
	onConnect,
	onDisconnect,
}: {
	query: string;
	onQueryChange: (value: string) => void;
	onClose: () => void;
	popular: ProviderSummary[];
	others: ProviderSummary[];
	connectedIds: Set<string>;
	disconnectingProviderId: string | null;
	providerErrors: Record<string, string>;
	onConnect: (provider: ProviderSummary) => void;
	onDisconnect: (providerId: string) => Promise<void>;
}) {
	const empty = popular.length === 0 && others.length === 0;

	return (
		<>
			<StepHeader title="Connect provider" onClose={onClose} />
			<div className="grid min-h-0 grid-rows-[auto,minmax(0,1fr)] px-4 pb-4 pt-3 sm:px-4">
				<div className="border-b border-border/40 px-1 pb-3">
					<p className="text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground/80">Workspace providers</p>
					<div className="mt-2 flex items-end justify-between gap-4">
						<div className="max-w-xl space-y-1">
							<h2 className="text-xl font-semibold tracking-tight text-foreground">Connect the models your workspace can use</h2>
							<p className="text-sm leading-5 text-muted-foreground">
								Keep provider setup light and editable. Add a key, finish OAuth, or remove access without leaving this sheet.
							</p>
						</div>
						<Badge variant="outline" className="hidden rounded-full border-border/60 px-2.5 py-1 text-[11px] font-medium text-muted-foreground sm:inline-flex">
							{connectedIds.size} connected
						</Badge>
					</div>
				</div>

				<div className="relative px-1 pb-3 pt-3">
					<Search className="pointer-events-none absolute left-5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						value={query}
						onChange={(event) => onQueryChange(event.target.value)}
						placeholder="Search providers, IDs, or notes"
						className="h-9 rounded-full border-border/50 bg-muted/20 pl-10 text-sm shadow-none"
						autoFocus
					/>
				</div>

				<div className="min-h-0 overflow-y-auto px-1 pb-2">

					{popular.length > 0 && (
						<ProviderListGroup
							label="Popular"
							providers={popular}
							connectedIds={connectedIds}
							disconnectingProviderId={disconnectingProviderId}
							providerErrors={providerErrors}
							onConnect={onConnect}
							onDisconnect={onDisconnect}
						/>
					)}

					{others.length > 0 && (
						<ProviderListGroup
							label="Other"
							providers={others}
							connectedIds={connectedIds}
							disconnectingProviderId={disconnectingProviderId}
							providerErrors={providerErrors}
							onConnect={onConnect}
							onDisconnect={onDisconnect}
						/>
					)}

					{empty && <p className="rounded-2xl border border-dashed border-border/60 px-4 py-12 text-center text-sm text-muted-foreground">No providers found.</p>}
				</div>
			</div>
		</>
	);
}

function ProviderListGroup({
	label,
	providers,
	connectedIds,
	disconnectingProviderId,
	providerErrors,
	onConnect,
	onDisconnect,
}: {
	label: string;
	providers: ProviderSummary[];
	connectedIds: Set<string>;
	disconnectingProviderId: string | null;
	providerErrors: Record<string, string>;
	onConnect: (provider: ProviderSummary) => void;
	onDisconnect: (providerId: string) => Promise<void>;
}) {
	return (
		<div className="mt-4">
			<div className="flex items-center justify-between px-3 py-1.5">
				<p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground/80">{label}</p>
				<p className="text-xs text-muted-foreground">{providers.length}</p>
			</div>
			<div className="overflow-hidden rounded-xl border border-border/50 bg-background/70">
				{providers.map((provider) => (
					<ProviderListRow
						key={provider.id}
						provider={provider}
						connected={connectedIds.has(provider.id)}
						disconnecting={disconnectingProviderId === provider.id}
						error={providerErrors[provider.id]}
						onConnect={() => onConnect(provider)}
						onDisconnect={() => void onDisconnect(provider.id)}
					/>
				))}
			</div>
		</div>
	);
}

function ProviderListRow({
	provider,
	connected,
	disconnecting,
	error,
	onConnect,
	onDisconnect,
}: {
	provider: ProviderSummary;
	connected: boolean;
	disconnecting: boolean;
	error?: string;
	onConnect: () => void;
	onDisconnect: () => void;
}) {
	const description = DESCRIPTIONS[provider.id];
	const badge = BADGES[provider.id];

	return (
		<div className="border-b border-border/40 last:border-b-0">
			<div className="flex items-center gap-3 px-3 py-2.5 transition-colors hover:bg-muted/20">
				<div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl border border-border/50 bg-muted/30">
					<span className="text-sm font-semibold uppercase text-muted-foreground">{provider.name.slice(0, 1)}</span>
				</div>
				<div className="min-w-0 flex-1 space-y-1">
					<div className="flex items-center gap-2">
						<span className="truncate text-sm font-medium text-foreground">{provider.name}</span>
						{badge && (
							<Badge variant="outline" className="shrink-0 rounded-full border-border/60 px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
								{badge}
							</Badge>
						)}
						{connected && (
							<Badge className="shrink-0 rounded-full bg-emerald-500/12 px-2 py-0.5 text-[10px] font-medium text-emerald-700 hover:bg-emerald-500/12 dark:text-emerald-300">
								Connected
							</Badge>
						)}
					</div>
					{description ? (
						<p className="line-clamp-2 text-[13px] leading-5 text-muted-foreground">{description}</p>
					) : (
						<p className="text-[13px] leading-5 text-muted-foreground">Use this provider inside your OpenCode workspace.</p>
					)}
				</div>
				{connected ? (
					<div className="flex items-center gap-2">
						<Button
							variant="ghost"
							size="sm"
							className="h-8 rounded-full px-3 text-muted-foreground hover:bg-destructive/5 hover:text-destructive"
							onClick={onDisconnect}
							disabled={disconnecting}
						>
							{disconnecting ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <Trash2 className="mr-1 h-3.5 w-3.5" />}
							Remove
						</Button>
					</div>
				) : (
					<Button variant="ghost" size="sm" className="h-8 rounded-full px-3 hover:bg-primary/8" onClick={onConnect}>
						<Plus className="mr-1 h-3.5 w-3.5" />
						Connect
					</Button>
				)}
			</div>
			{error && <p className="px-4 pb-3 text-xs text-destructive">{error}</p>}
		</div>
	);
}

function ProviderMethodStepView({
	provider,
	methods,
	selectedMethodIndex,
	apiKey,
	error,
	saving,
	authorizing,
	onBack,
	onClose,
	onSelectMethod,
	onApiKeyChange,
	onContinue,
	continueDisabled,
}: {
	provider: ActiveProviderState;
	methods: ProviderMethodEntry[];
	selectedMethodIndex: number | null;
	apiKey: string;
	error?: string;
	saving: boolean;
	authorizing: boolean;
	onBack: () => void;
	onClose: () => void;
	onSelectMethod: (methodIndex: number) => void;
	onApiKeyChange: (value: string) => void;
	onContinue: () => void;
	continueDisabled: boolean;
}) {
	const selectedMethod = methods.find((method) => method.methodIndex === selectedMethodIndex) ?? null;
	const showApiInput = selectedMethod?.type === "api";

	return (
		<>
			<StepHeader title={`Connect ${provider.name}`} onBack={onBack} onClose={onClose} />
			<div className="min-h-0 overflow-y-auto px-4 py-4 sm:px-5">
				<div className="mx-auto flex max-w-2xl flex-col gap-4">
					<div className="flex flex-col gap-4 border-b border-border/40 pb-5 sm:flex-row sm:items-end sm:justify-between">
						<div className="flex items-center gap-4">
							<div className="flex h-10 w-10 items-center justify-center rounded-xl border border-border/50 bg-muted/30">
							<span className="text-lg font-semibold uppercase text-muted-foreground">{provider.name.slice(0, 1)}</span>
						</div>
							<div>
								<p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground/80">Provider details</p>
								<h2 className="mt-1 text-xl font-semibold tracking-tight">Connect {provider.name}</h2>
								<p className="mt-1 text-sm leading-5 text-muted-foreground">Select a compact authentication method and keep the setup easy to edit later.</p>
							</div>
						</div>
						<Badge variant="outline" className="w-fit rounded-full border-border/60 px-2.5 py-1 text-[11px] font-medium text-muted-foreground">{methods.length} method{methods.length === 1 ? "" : "s"}</Badge>
					</div>

					<div className="rounded-2xl border border-border/50 bg-muted/10 p-3 sm:p-4">
						<div className="space-y-2">
							{methods.map((method) => {
								const selected = method.methodIndex === selectedMethodIndex;
								return (
									<button
										key={method.methodIndex}
										type="button"
										onClick={() => onSelectMethod(method.methodIndex)}
										className={cn(
											"flex w-full items-center gap-3 rounded-xl border px-3.5 py-3 text-left transition-colors",
											selected
												? "border-primary/40 bg-background shadow-sm"
												: "border-transparent bg-transparent hover:border-border/50 hover:bg-background/80",
										)}
									>
										<span className={cn("flex h-5 w-5 items-center justify-center rounded-full border", selected ? "border-primary bg-primary text-primary-foreground" : "border-border/60 bg-background")}> 
											{selected && <span className="h-2 w-2 rounded-[2px] bg-current" />}
										</span>
										<div className="flex-1">
											<p className="text-sm font-medium tracking-tight text-foreground">{method.type === "api" ? "API key" : method.label}</p>
											<p className="mt-0.5 text-[13px] text-muted-foreground">{method.type === "api" ? `Paste a ${provider.name} key directly.` : "Authorize in a browser window and return here."}</p>
										</div>
									</button>
								);
							})}
						</div>

						{showApiInput && (
							<div className="mt-3 space-y-2 rounded-xl border border-border/40 bg-background/80 p-3">
								<p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground/80">API key</p>
								<p className="text-sm text-muted-foreground">Enter your {provider.name} API key. You can rotate or remove it later.</p>
								<Input
									value={apiKey}
									onChange={(event) => onApiKeyChange(event.target.value)}
									onKeyDown={(event) => event.key === "Enter" && onContinue()}
									placeholder={`Paste ${provider.name} API key...`}
									type="password"
									className="h-10 rounded-xl border-border/60 bg-background font-mono text-sm shadow-none"
									autoFocus
								/>
							</div>
						)}
					</div>

					{error && <p className="rounded-2xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}

					<div className="flex justify-end">
						<Button size="sm" className="min-w-28 rounded-full px-5" disabled={continueDisabled} onClick={onContinue}>
							{saving || authorizing ? <Loader2 className="h-4 w-4 animate-spin" /> : getMethodButtonLabel(selectedMethod)}
						</Button>
					</div>
				</div>
			</div>
		</>
	);
}

function OAuthStepView({
	providerName,
	authorization,
	onBack,
	onClose,
	onFinish,
}: {
	providerName: string;
	authorization: ProviderAuthAuthorization | null;
	onBack: () => void;
	onClose: () => void;
	onFinish: (code?: string) => Promise<void>;
}) {
	const [code, setCode] = useState("");
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [copied, setCopied] = useState(false);

	useEffect(() => {
		if (!authorization || authorization.method !== "auto") return;
		let active = true;
		setLoading(true);
		setError(null);
		void onFinish()
			.catch((finishError) => {
				if (!active) return;
				setError(getErrorMessage(finishError, "Authorization failed. Please try again."));
			})
			.finally(() => {
				if (active) setLoading(false);
			});
		return () => {
			active = false;
		};
	}, [authorization, onFinish]);

	useEffect(() => {
		if (!copied) return;
		const timeoutId = window.setTimeout(() => setCopied(false), 1500);
		return () => window.clearTimeout(timeoutId);
	}, [copied]);

	const handleSubmitCode = async () => {
		if (!code.trim()) return;
		setLoading(true);
		setError(null);
		try {
			await onFinish(code.trim());
		} catch (finishError) {
			setError(getErrorMessage(finishError, "Authentication failed"));
		} finally {
			setLoading(false);
		}
	};

	const handleCopy = async () => {
		if (!authorization) return;
		await navigator.clipboard.writeText(getAutoInstructionText(authorization));
		setCopied(true);
	};

	return (
		<>
			<StepHeader title={`Connect ${providerName}`} onBack={onBack} onClose={onClose} />
			<div className="min-h-0 overflow-y-auto px-4 py-4 sm:px-5">
				<div className="mx-auto flex max-w-2xl flex-col gap-4">
					<div className="flex items-center gap-4 border-b border-border/40 pb-5">
						<div className="flex h-10 w-10 items-center justify-center rounded-xl border border-border/50 bg-muted/30">
							<span className="text-lg font-semibold uppercase text-muted-foreground">{providerName.slice(0, 1)}</span>
						</div>
						<div>
							<p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground/80">Authorization</p>
							<h2 className="mt-1 text-xl font-semibold tracking-tight">Connect {providerName}</h2>
						</div>
					</div>

					{!authorization ? (
						<div className="flex min-h-52 items-center justify-center rounded-2xl border border-dashed border-border/60 bg-muted/10">
							<Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
						</div>
					) : (
						<div className="space-y-4 rounded-2xl border border-border/50 bg-muted/10 p-4">
							<p className="max-w-2xl text-sm leading-6 text-muted-foreground">
								Visit{" "}
								<button type="button" className="underline underline-offset-4" onClick={() => window.open(authorization.url, "_blank", "noopener,noreferrer")}>
									this link
								</button>{" "}
								and {authorization.method === "code" ? "enter the code below" : "complete the authorization in your browser"} to connect your account and use {providerName} models in OpenCode.
							</p>

							<div>
								<Button variant="outline" size="sm" className="gap-2 rounded-full" onClick={() => window.open(authorization.url, "_blank", "noopener,noreferrer")}>
									<ExternalLink className="h-4 w-4" />
									Open authorization page
								</Button>
							</div>

							{authorization.method === "code" ? (
								<div className="space-y-3 rounded-xl border border-border/40 bg-background/80 p-3">
									<p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground/80">Authorization code</p>
									<div className="flex gap-2">
										<Input
											value={code}
											onChange={(event) => setCode(event.target.value)}
											onKeyDown={(event) => event.key === "Enter" && void handleSubmitCode()}
											placeholder="Paste authorization code..."
											className="h-10 rounded-xl border-border/60 bg-background font-mono text-sm"
											autoFocus
										/>
										<Button size="sm" className="rounded-full px-5" disabled={!code.trim() || loading} onClick={() => void handleSubmitCode()}>
											{loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Submit"}
										</Button>
									</div>
								</div>
							) : (
								<div className="space-y-4">
									<div className="space-y-3 rounded-xl border border-border/40 bg-background/80 p-3">
										<p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground/80">Confirmation status</p>
										<div className="relative">
											<Input
												readOnly
												value={getAutoInstructionText(authorization)}
												className="h-11 rounded-2xl border-border/60 bg-background pr-14 font-mono text-sm"
											/>
											<Button
												variant="ghost"
												size="icon"
												className="absolute right-1.5 top-1/2 h-8 w-8 -translate-y-1/2 rounded-xl"
												onClick={() => void handleCopy()}
											>
												{copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
											</Button>
										</div>
									</div>

									<div className="flex items-center gap-3 rounded-2xl border border-dashed border-border/60 bg-muted/10 px-4 py-3 text-sm text-muted-foreground">
										<Loader2 className="h-4 w-4 animate-spin" />
										<span>{loading ? "Waiting for authorization..." : "Authorization finished."}</span>
									</div>
								</div>
							)}

							{error && <p className="rounded-2xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}
						</div>
					)}
				</div>
			</div>
		</>
	);
}

function StepHeader({ title, onBack, onClose }: { title: string; onBack?: () => void; onClose: () => void }) {
	return (
		<div className="flex items-center justify-between border-b border-border/40 px-4 py-2.5 sm:px-4">
			<div className="flex items-center gap-2">
				{onBack ? (
					<Button variant="ghost" size="icon" className="h-7 w-7 rounded-full" onClick={onBack}>
						<ArrowLeft className="h-4 w-4" />
					</Button>
				) : (
					<div className="h-7 w-7" />
				)}
				<div>
					<p className="text-[10px] font-medium uppercase tracking-[0.16em] text-muted-foreground/80">OpenCode</p>
					<h1 className="text-sm font-medium tracking-tight text-foreground">{title}</h1>
				</div>
			</div>
			<Button variant="ghost" size="icon" className="h-7 w-7 rounded-full text-muted-foreground" onClick={onClose}>
				<X className="h-4 w-4" />
			</Button>
		</div>
	);
}
