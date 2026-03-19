import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";
import { useDocs } from "../../contexts/DocsContext";

interface AppBreadcrumbProps {
	currentPage: string;
	projectName: string;
}

const pageLabels: Record<string, string> = {
	dashboard: "Dashboard",
	kanban: "Kanban",
	tasks: "Tasks",
	docs: "Docs",
	templates: "Templates",
	imports: "Imports",
	chat: "AI Chat",
	workspaces: "AI Workspaces",
	config: "Settings",
};

function BreadcrumbSeparator() {
	return <ChevronRight className="w-3.5 h-3.5 text-muted-foreground/60 shrink-0" />;
}

function BreadcrumbLink({
	to,
	onClick,
	children,
}: {
	to: string;
	onClick?: (e: React.MouseEvent) => void;
	children: React.ReactNode;
}) {
	return (
		<Link
			to={to}
			onClick={onClick}
			className="text-muted-foreground hover:text-foreground transition-colors truncate"
		>
			{children}
		</Link>
	);
}

function BreadcrumbCurrent({ children }: { children: React.ReactNode }) {
	return (
		<span className="text-foreground font-medium truncate">{children}</span>
	);
}

function DocsBreadcrumbSegments() {
	const { selectedDoc, currentFolder, navigateToFolder } = useDocs();

	if (selectedDoc) {
		// Viewing a doc: show folder path (clickable) + doc title
		const segments: React.ReactNode[] = [];

		if (selectedDoc.folder) {
			const folderParts = selectedDoc.folder.split("/");
			for (let i = 0; i < folderParts.length; i++) {
				const part = folderParts[i]!;
				const folderPath = folderParts.slice(0, i + 1).join("/");
				segments.push(
					<BreadcrumbSeparator key={`sep-${folderPath}`} />,
					<BreadcrumbLink
						key={`folder-${folderPath}`}
						to={`/docs/${folderPath}`}
						onClick={(e) => {
							e.preventDefault();
							navigateToFolder(folderPath);
						}}
					>
						{part}
					</BreadcrumbLink>,
				);
			}
		}

		segments.push(
			<BreadcrumbSeparator key="sep-doc" />,
			<BreadcrumbCurrent key="doc">
				{selectedDoc.metadata.title}
			</BreadcrumbCurrent>,
		);

		return <>{segments}</>;
	}

	if (currentFolder) {
		// Viewing a folder: show folder path segments (last one current)
		const parts = currentFolder.split("/");
		const segments: React.ReactNode[] = [];

		for (let i = 0; i < parts.length; i++) {
			const part = parts[i]!;
			const folderPath = parts.slice(0, i + 1).join("/");
			const isLast = i === parts.length - 1;
			segments.push(
				<BreadcrumbSeparator key={`sep-${folderPath}`} />,
				isLast ? (
					<BreadcrumbCurrent key={`folder-${folderPath}`}>
						{part}
					</BreadcrumbCurrent>
				) : (
					<BreadcrumbLink
						key={`folder-${folderPath}`}
						to={`/docs/${folderPath}`}
						onClick={(e) => {
							e.preventDefault();
							navigateToFolder(folderPath);
						}}
					>
						{part}
					</BreadcrumbLink>
				),
			);
		}

		return <>{segments}</>;
	}

	return null;
}

export function AppBreadcrumb({ currentPage, projectName }: AppBreadcrumbProps) {
	const pageLabel = pageLabels[currentPage] || "Dashboard";

	return (
		<nav className="flex items-center gap-1.5 text-sm min-w-0 flex-1">
			<BreadcrumbLink to="/">{projectName}</BreadcrumbLink>
			{currentPage !== "dashboard" && (
				<>
					<BreadcrumbSeparator />
					{currentPage === "docs" ? (
						<>
							<BreadcrumbLink to="/docs">Docs</BreadcrumbLink>
							<DocsBreadcrumbSegments />
						</>
					) : (
						<BreadcrumbCurrent>{pageLabel}</BreadcrumbCurrent>
					)}
				</>
			)}
		</nav>
	);
}
