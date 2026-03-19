/**
 * Slash menu data loaded from OpenCode's unified command registry.
 */

import { useEffect, useState } from "react";

import { opencodeApi, type OpenCodeCommandDefinition } from "../api/client";

export interface WorkflowShortcut {
	name: string;
	description: string;
	usage: string;
	example?: string;
	source: "skill";
	template?: string;
	hints?: string[];
	subtask?: boolean;
}

export interface SlashAction {
	name: string;
	description: string;
	usage: string;
	example?: string;
	source: "command";
	template?: string;
	hints?: string[];
	subtask?: boolean;
}

export type SlashItem = WorkflowShortcut | SlashAction;

function normalizeSlashName(name: string | undefined): string {
	if (!name) return "/";
	return name.startsWith("/") ? name : `/${name}`;
}

function normalizeSlashItem(item: OpenCodeCommandDefinition | any): SlashItem {
	const source = item?.source === "skill" ? "skill" : "command";
	return {
		name: normalizeSlashName(item?.name),
		description: item?.description || "",
		usage: normalizeSlashName(item?.name),
		example: undefined,
		source,
		template: typeof item?.template === "string" ? item.template : undefined,
		hints: Array.isArray(item?.hints)
			? item.hints.filter((hint: unknown): hint is string => typeof hint === "string")
			: undefined,
		subtask: Boolean(item?.subtask),
	};
}

export function useSlashItems(directory?: string | null) {
	const [slashItems, setSlashItems] = useState<SlashItem[]>([]);
	const [loading, setLoading] = useState(true);

	useEffect(() => {
		setLoading(true);
		opencodeApi
			.listCommands(directory)
			.then((data) => {
				setSlashItems((Array.isArray(data) ? data : []).map(normalizeSlashItem));
			})
			.catch((err) => {
				console.error("Failed to load slash items:", err);
				setSlashItems([]);
			})
			.finally(() => setLoading(false));
	}, [directory]);

	return { slashItems, loading };
}
