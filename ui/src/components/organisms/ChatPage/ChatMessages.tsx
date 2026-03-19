import { Loader2 } from "lucide-react";

import { ChatThread } from "../../chat/ChatThread";
import type { ChatComposerFile, ChatSession } from "../../../models/chat";
import { ChatWelcome } from "./ChatWelcome";

interface ChatMessagesProps {
	session: ChatSession;
	loading?: boolean;
	onSend: (message: string, files?: ChatComposerFile[]) => void;
	onSubmitQuestion: (messageId: string, blockId: string, answers: string[][]) => Promise<void> | void;
	onRejectQuestion: (messageId: string, blockId: string) => Promise<void> | void;
	onRevert?: (messageId: string) => void;
	onPreviewTask?: (taskId: string) => void;
	onPreviewDoc?: (docPath: string) => void;
}

export function ChatMessages({
	session,
	loading,
	onSend,
	onSubmitQuestion,
	onRejectQuestion,
	onRevert,
	onPreviewTask,
	onPreviewDoc,
}: ChatMessagesProps) {
	if (loading && session.messages.length === 0) {
		return (
			<div className="flex flex-1 items-center justify-center">
				<Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
			</div>
		);
	}

	if (session.messages.length === 0 && session.status !== "streaming") {
		return <ChatWelcome onSend={onSend} />;
	}

	return (
		<ChatThread
			session={session}
			onSubmitQuestion={onSubmitQuestion}
			onRejectQuestion={onRejectQuestion}
			onRevert={onRevert}
			onPreviewTask={onPreviewTask}
			onPreviewDoc={onPreviewDoc}
		/>
	);
}
