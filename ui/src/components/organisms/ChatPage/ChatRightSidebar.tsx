import type { RefObject } from "react";
import { Bot, PanelRight } from "lucide-react";

import type { ChatSession } from "../../../models/chat";
import { SubAgentBlock } from "../../chat/SubAgentBlock";

interface ChatRightSidebarProps {
	subSessions: ChatSession[];
	subAgentSectionRef?: RefObject<HTMLDivElement | null>;
}

export function ChatRightSidebar({
	subSessions,
	subAgentSectionRef,
}: ChatRightSidebarProps) {
	return (
		<aside className="hidden w-[300px] shrink-0 border-l border-border/70 bg-muted/10 lg:flex lg:flex-col xl:w-[328px]">
			<div className="flex items-center gap-2 border-b border-border/70 px-3 py-2.5">
				<div className="flex h-7 w-7 items-center justify-center rounded-xl bg-primary/10 text-primary">
					<PanelRight className="h-4 w-4" />
				</div>
				<div>
					<div className="text-sm font-semibold text-foreground">Session Sidebar</div>
					<div className="text-xs text-muted-foreground">Context and agent activity</div>
				</div>
			</div>

			<div className="flex-1 space-y-3 overflow-y-auto p-3">
				<section ref={subAgentSectionRef} className="space-y-3">
					<div className="flex items-center gap-2 px-1">
						<Bot className="h-4 w-4 text-primary" />
						<h3 className="text-sm font-semibold text-foreground">
							Sub-agent{subSessions.length !== 1 ? "s" : ""}
						</h3>
						{Boolean(subSessions.length) && (
							<span className="rounded-md border border-border bg-background px-2 py-0.5 text-[11px] text-muted-foreground">
								{subSessions.length}
							</span>
						)}
					</div>

					{subSessions.length === 0 ? (
						<div className="rounded-2xl border border-dashed border-border/70 bg-background/70 px-3 py-5 text-sm text-muted-foreground">
							No sub-agent activity in this session yet.
						</div>
					) : (
						subSessions.map((subSession) => (
							<SubAgentBlock
								key={subSession.id}
								session={subSession}
								indented={false}
							/>
						))
					)}
				</section>
			</div>
		</aside>
	);
}
