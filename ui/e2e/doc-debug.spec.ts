import { test, expect } from "@playwright/test";
import { startServer, type TestServer } from "./helpers";

let server: TestServer;
test.beforeAll(async () => { server = await startServer(); });
test.afterAll(() => { server?.cleanup(); });

test("debug doc creation", async ({ page }) => {
	const logs: string[] = [];
	const responses: string[] = [];
	
	page.on("console", (msg) => logs.push(`[${msg.type()}] ${msg.text()}`));
	page.on("response", (resp) => responses.push(`${resp.status()} ${resp.url()}`));
	page.on("pageerror", (err) => logs.push(`[PAGE_ERROR] ${err.message}`));

	await page.goto(`${server.baseURL}/docs`);
	await page.waitForTimeout(1000);

	// Click New Doc
	const newBtn = page.locator("button").filter({ hasText: "New Doc" }).first();
	await newBtn.click();
	await page.waitForTimeout(500);

	// Fill title
	await page.locator('input[placeholder="Untitled"]').fill("Debug Doc");
	await page.waitForTimeout(200);

	// Click Create
	const createBtn = page.locator("button").filter({ hasText: "Create" }).first();
	console.log("Create btn visible:", await createBtn.isVisible());
	console.log("Create btn disabled:", await createBtn.isDisabled());
	
	// Clear response log before clicking
	responses.length = 0;
	await createBtn.click();
	await page.waitForTimeout(3000);

	// Check what happened
	console.log("=== RESPONSES AFTER CLICK ===");
	for (const r of responses) {
		if (r.includes("/api/")) console.log(r);
	}
	console.log("=== CONSOLE LOGS ===");
	for (const l of logs) {
		if (l.includes("error") || l.includes("Error") || l.includes("fail") || l.includes("PAGE_ERROR")) {
			console.log(l);
		}
	}

	// Check current page state
	const hasNewDoc = await page.locator("button").filter({ hasText: "New Doc" }).isVisible().catch(() => false);
	const hasCreate = await page.locator("button").filter({ hasText: "Create" }).isVisible().catch(() => false);
	console.log("New Doc btn visible (back to list?):", hasNewDoc);
	console.log("Create btn still visible (still on form?):", hasCreate);
	
	// Take screenshot
	await page.screenshot({ path: "test-results/doc-debug.png" });
});
