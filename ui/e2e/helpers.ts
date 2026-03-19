/**
 * E2E test helpers — start/stop knowns server with isolated project
 */

import { execSync, spawn, type ChildProcess } from "node:child_process";
import { mkdtempSync, rmSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const BINARY = process.env.TEST_BINARY
	? resolve(process.cwd(), process.env.TEST_BINARY)
	: resolve(__dirname, "../../bin/knowns");

/** Find an available port */
function findPort(): number {
	// Use a random port in the high range
	return 10000 + Math.floor(Math.random() * 50000);
}

/** Wait for HTTP server to be ready */
async function waitForServer(
	url: string,
	timeoutMs = 15000,
): Promise<void> {
	const start = Date.now();
	while (Date.now() - start < timeoutMs) {
		try {
			const res = await fetch(url);
			if (res.ok) return;
		} catch {
			// not ready yet
		}
		await new Promise((r) => setTimeout(r, 200));
	}
	throw new Error(`Server not ready at ${url} after ${timeoutMs}ms`);
}

export interface TestServer {
	/** Base URL of the running server */
	baseURL: string;
	/** Port the server is listening on */
	port: number;
	/** Path to the temporary project directory */
	projectDir: string;
	/** Run a CLI command against the test project */
	cli: (args: string) => string;
	/** Stop the server and clean up */
	cleanup: () => void;
}

/**
 * Start a knowns server with an isolated temporary project.
 * Returns helpers for interacting with the server.
 */
export async function startServer(): Promise<TestServer> {
	// Verify binary exists
	if (!existsSync(BINARY)) {
		throw new Error(
			`Binary not found at ${BINARY}. Run 'make build' first.`,
		);
	}

	// Create isolated project directory
	const projectDir = mkdtempSync(join(tmpdir(), "knowns-e2e-"));

	// Initialize git + knowns project
	execSync("git init", { cwd: projectDir, stdio: "ignore" });
	execSync("git config user.email test@test.com", {
		cwd: projectDir,
		stdio: "ignore",
	});
	execSync("git config user.name Test", {
		cwd: projectDir,
		stdio: "ignore",
	});
	execSync(`${BINARY} init "E2E Test Project" --no-wizard --no-open`, {
		cwd: projectDir,
		stdio: "ignore",
	});

	// Find available port and start server
	const port = findPort();
	const serverProcess: ChildProcess = spawn(
		BINARY,
		["browser", "--port", String(port), "--no-open"],
		{
			cwd: projectDir,
			stdio: ["ignore", "pipe", "pipe"],
			env: { ...process.env, NO_COLOR: "1" },
		},
	);

	const baseURL = `http://localhost:${port}`;

	// Wait for server to be ready
	try {
		await waitForServer(baseURL);
	} catch (err) {
		serverProcess.kill("SIGTERM");
		rmSync(projectDir, { recursive: true, force: true });
		throw err;
	}

	const cli = (args: string): string => {
		return execSync(`${BINARY} ${args}`, {
			cwd: projectDir,
			encoding: "utf-8",
			timeout: 10000,
			env: { ...process.env, NO_COLOR: "1" },
		}).trim();
	};

	const cleanup = () => {
		serverProcess.kill("SIGTERM");
		// Give process time to exit, then force kill
		setTimeout(() => {
			try {
				serverProcess.kill("SIGKILL");
			} catch {
				// already dead
			}
		}, 2000);
		rmSync(projectDir, { recursive: true, force: true });
	};

	return { baseURL, port, projectDir, cli, cleanup };
}
