import { Check, Loader2, Plus, Search, Trash2, X } from "lucide-react";
import { useMemo, useState } from "react";

import type { ProviderAuthAuthorization } from "../../../api/client";
import { useOpenCode } from "../../../contexts/OpenCodeContext";
import { cn } from "../../../lib/utils";
import { Badge } from "../../ui/badge";
import { Button } from "../../ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "../../ui/dialog";
import { Input } from "../../ui/input";
import { OpenCodeProviderIcon } from "../OpenCodeProviderIcon";
import { ProviderOAuthDialog } from "./ProviderOAuthDialog";

// ── Static metadata for known providers ────────────────────────────

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

// ── Dialog ──────────────────────────────────────────────────────────

interface ProviderConnectDialogProps {
	open: boolean;
	onClose: () => void;
}

export function ProviderConnectDialog({ open, onClose }: ProviderConnectDialogProps) {
	const { providerResponse, providerAuthMethods, connectWithApiKey, startOAuth, finishOAuth, disconnectProvider } = useOpenCode();
	const [query, setQuery] = useState("");
	const [expandedId, setExpandedId] = useState<string | null>(null);
	const [apiKey, setApiKey] = useState("");
	const [saving, setSaving] = useState(false);
	const [oauthState, setOauthState] = useState<{
		id: string;
		name: string;
		authorization: ProviderAuthAuthorization | null;
	} | null>(null);

	const connectedIds = useMemo(() => new Set(providerResponse?.connected ?? []), [providerResponse]);

	const filtered = useMemo(() => {
		const q = query.trim().toLowerCase();
		const all = providerResponse?.all ?? [];
		if (!q) return all;
		return all.filter(
			(p) =>
				p.name.toLowerCase().includes(q) ||
				p.id.toLowerCase().includes(q) ||
				DESCRIPTIONS[p.id]?.toLowerCase().includes(q),
		);
	}, [providerResponse, query]);

	const popular = filtered.filter((p) => POPULAR.has(p.id));
	const others = filtered.filter((p) => !POPULAR.has(p.id));

	const handleConnect = async (provider: { id: string; name: string }) => {
		const methods = providerAuthMethods?.[provider.id] ?? [];
		const hasOAuth = methods.some((m) => m.type === "oauth");
		const hasApi = methods.some((m) => m.type === "api");

		if (hasOAuth) {
			setOauthState({ id: provider.id, name: provider.name, authorization: null });
			try {
				const auth = await startOAuth(provider.id, 0);
				setOauthState((prev) => (prev ? { ...prev, authorization: auth } : null));
				if (auth.url) window.open(auth.url, "_blank", "noopener,noreferrer");
			} catch {
				setOauthState(null);
			}
		} else if (hasApi) {
			setExpandedId((prev) => (prev === provider.id ? null : provider.id));
			setApiKey("");
		}
	};

	const handleDisconnect = async (providerId: string) => {
		await disconnectProvider(providerId);
	};

	const handleSaveKey = async (providerId: string) => {
		if (!apiKey.trim()) return;
		setSaving(true);
		try {
			await connectWithApiKey(providerId, apiKey.trim());
			setExpandedId(null);
			setApiKey("");
		} finally {
			setSaving(false);
		}
	};

	return (
		<>
			<Dialog
				open={open && !oauthState}
				onOpenChange={(v) => {
					if (!v) {
						setExpandedId(null);
						setApiKey("");
						setQuery("");
						onClose();
					}
				}}
			>
				<DialogContent className="grid h-[80vh] max-h-[80vh] max-w-lg grid-rows-[auto,auto,minmax(0,1fr)] gap-0 overflow-hidden p-0">
					<DialogHeader className="border-b border-border/40 px-5 py-4">
						<DialogTitle className="text-base font-semibold">Connect provider</DialogTitle>
					</DialogHeader>

					{/* Search */}
					<div className="px-3 pt-3 pb-1">
						<div className="relative">
							<Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
							<Input
								value={query}
								onChange={(e) => setQuery(e.target.value)}
								placeholder="Search providers"
								className="h-10 rounded-xl border-border/50 bg-muted/30 pl-9"
								autoFocus
							/>
						</div>
					</div>

					{/* Provider list */}
					<div className="min-h-0 h-full overflow-y-auto px-1 pb-3">
						{popular.length > 0 && (
							<ProviderGroup
								label="Popular"
								providers={popular}
								connectedIds={connectedIds}
								expandedId={expandedId}
								apiKey={apiKey}
								saving={saving}
								onApiKeyChange={setApiKey}
								onConnect={handleConnect}
								onSaveKey={handleSaveKey}
								onCollapse={() => setExpandedId(null)}
								onDisconnect={handleDisconnect}
							/>
						)}
						{others.length > 0 && (
							<ProviderGroup
								label="Other"
								providers={others}
								connectedIds={connectedIds}
								expandedId={expandedId}
								apiKey={apiKey}
								saving={saving}
								onApiKeyChange={setApiKey}
								onConnect={handleConnect}
								onSaveKey={handleSaveKey}
								onCollapse={() => setExpandedId(null)}
								onDisconnect={handleDisconnect}
							/>
						)}
						{filtered.length === 0 && (
							<p className="px-4 py-8 text-center text-sm text-muted-foreground">No providers found.</p>
						)}
					</div>
				</DialogContent>
			</Dialog>

			{/* OAuth flow */}
			{oauthState && (
				<ProviderOAuthDialog
					open
					providerName={oauthState.name}
					authorization={oauthState.authorization}
					onFinish={async (code) => {
						await finishOAuth(oauthState.id, 0, code);
					}}
					onClose={() => {
						setOauthState(null);
						onClose();
					}}
				/>
			)}
		</>
	);
}

// ── Group ───────────────────────────────────────────────────────────

function ProviderGroup({
	label,
	providers,
	connectedIds,
	expandedId,
	apiKey,
	saving,
	onApiKeyChange,
	onConnect,
	onSaveKey,
	onCollapse,
	onDisconnect,
}: {
	label: string;
	providers: Array<{ id: string; name: string }>;
	connectedIds: Set<string>;
	expandedId: string | null;
	apiKey: string;
	saving: boolean;
	onApiKeyChange: (v: string) => void;
	onConnect: (p: { id: string; name: string }) => void;
	onSaveKey: (id: string) => void;
	onCollapse: () => void;
	onDisconnect: (id: string) => void;
}) {
	return (
		<div className="mt-2">
			<p className="px-4 py-1.5 text-xs font-medium text-muted-foreground">{label}</p>
			{providers.map((provider) => (
				<ProviderRow
					key={provider.id}
					provider={provider}
					connected={connectedIds.has(provider.id)}
					expanded={expandedId === provider.id}
					apiKey={apiKey}
					saving={saving}
					onApiKeyChange={onApiKeyChange}
					onConnect={() => onConnect(provider)}
					onSaveKey={() => onSaveKey(provider.id)}
					onCollapse={onCollapse}
					onDisconnect={() => void onDisconnect(provider.id)}
				/>
			))}
		</div>
	);
}

// ── Row ─────────────────────────────────────────────────────────────

function ProviderRow({
	provider,
	connected,
	expanded,
	apiKey,
	saving,
	onApiKeyChange,
	onConnect,
	onSaveKey,
	onCollapse,
	onDisconnect,
}: {
	provider: { id: string; name: string };
	connected: boolean;
	expanded: boolean;
	apiKey: string;
	saving: boolean;
	onApiKeyChange: (v: string) => void;
	onConnect: () => void;
	onSaveKey: () => void;
	onCollapse: () => void;
	onDisconnect: () => void;
}) {
	const description = DESCRIPTIONS[provider.id];
	const badge = BADGES[provider.id];

	return (
		<div className={cn("mx-1 rounded-xl transition-colors", expanded && "bg-muted/40")}>
			<div
				className={cn(
					"flex cursor-default items-center gap-3 rounded-xl px-3 py-2.5 transition-colors",
					!expanded && "hover:bg-muted/50",
				)}
			>
				<OpenCodeProviderIcon providerName={provider.name} providerID={provider.id} size="sm" />
				<div className="flex min-w-0 flex-1 items-baseline gap-2">
					<span className="shrink-0 text-sm font-semibold">{provider.name}</span>
					{description && (
						<span className="truncate text-sm text-muted-foreground">{description}</span>
					)}
				</div>
				{badge && (
					<Badge variant="outline" className="shrink-0 text-xs">
						{badge}
					</Badge>
				)}
				{connected ? (
					<div className="flex items-center gap-1">
						<Check className="h-4 w-4 shrink-0 text-emerald-500" />
						<Button
							variant="ghost"
							size="icon"
							className="h-7 w-7 shrink-0 rounded-lg text-muted-foreground hover:text-destructive"
							onClick={onDisconnect}
							title="Disconnect provider"
						>
							<Trash2 className="h-3.5 w-3.5" />
						</Button>
					</div>
				) : (
					<Button
						variant="ghost"
						size="icon"
						className="h-7 w-7 shrink-0 rounded-lg text-muted-foreground hover:text-foreground"
						onClick={onConnect}
					>
						<Plus className="h-4 w-4" />
					</Button>
				)}
			</div>

			{/* Inline API key input */}
			{expanded && (
				<div className="px-3 pb-3">
					<div className="flex gap-2">
						<Input
							value={apiKey}
							onChange={(e) => onApiKeyChange(e.target.value)}
							onKeyDown={(e) => e.key === "Enter" && onSaveKey()}
							placeholder={`Paste ${provider.name} API key…`}
							type="password"
							className="h-8 flex-1 font-mono text-xs"
							autoFocus
						/>
						<Button
							size="sm"
							className="h-8 px-4 text-xs"
							disabled={!apiKey.trim() || saving}
							onClick={onSaveKey}
						>
							{saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Connect"}
						</Button>
						<Button variant="ghost" size="icon" className="h-8 w-8" onClick={onCollapse}>
							<X className="h-3.5 w-3.5" />
						</Button>
					</div>
				</div>
			)}
		</div>
	);
}
