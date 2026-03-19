import { useState, type ReactNode } from "react";
import { Settings2 } from "lucide-react";

import type { OpenCodeCatalogState } from "../../models/chat";
import { Button } from "../ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
} from "../ui/dialog";
import { OpenCodeModelManager } from "./OpenCodeModelManager";

interface OpenCodeModelManagerDialogProps {
	catalog: OpenCodeCatalogState;
	lastLoadedAt?: string | null;
	onSetDefaultModel: (modelKey: string | null) => void;
	onUpdateModelPref: (modelKey: string, patch: { enabled?: boolean; pinned?: boolean }) => void;
	onToggleProviderHidden?: (providerID: string, hidden: boolean, modelKeys?: string[]) => void;
	showProviderVisibility?: boolean;
	triggerIcon?: ReactNode;
}

export function OpenCodeModelManagerDialog({
	catalog,
	lastLoadedAt,
	onSetDefaultModel,
	onUpdateModelPref,
	onToggleProviderHidden,
	showProviderVisibility = false,
	triggerIcon,
}: OpenCodeModelManagerDialogProps) {
	const [open, setOpen] = useState(false);

	return (
		<Dialog open={open} onOpenChange={setOpen}>
			<DialogTrigger asChild>
				<Button
					variant="outline"
					size="icon"
					title="Manage models"
					aria-label="Manage models"
					className="h-8 w-8 rounded-lg border-border/60 bg-background/80"
				>
					{triggerIcon || <Settings2 className="h-4 w-4" />}
				</Button>
			</DialogTrigger>
			<DialogContent className="grid h-[80vh] max-h-[80vh] max-w-3xl grid-rows-[auto,minmax(0,1fr)] overflow-hidden p-0">
				<div className="border-b border-border/40 px-6 py-4">
					<DialogHeader className="space-y-1 text-left">
						<DialogTitle className="text-lg font-semibold">Manage models</DialogTitle>
						<DialogDescription className="text-sm text-muted-foreground">
							Choose which providers and models appear in the picker.
						</DialogDescription>
					</DialogHeader>
				</div>
				<div className="min-h-0 overflow-hidden px-6 py-4">
					<OpenCodeModelManager
						catalog={catalog}
						lastLoadedAt={lastLoadedAt}
						onSetDefaultModel={onSetDefaultModel}
						onUpdateModelPref={onUpdateModelPref}
						onToggleProviderHidden={onToggleProviderHidden}
						showProviderVisibility={showProviderVisibility}
						compact
					/>
				</div>
			</DialogContent>
		</Dialog>
	);
}
