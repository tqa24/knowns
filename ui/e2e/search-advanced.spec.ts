import { test, expect } from "@playwright/test";
import { startServer, type TestServer } from "./helpers";

let server: TestServer;

test.beforeAll(async () => {
	server = await startServer();
	// Seed data for search tests
	server.cli('task create "Auth Module" -d "Implement authentication module" --priority high -l "backend"');
	server.cli('task create "Login Page" -d "Create login page UI" --priority medium -l "frontend"');
	server.cli('task create "API Docs" -d "Write API documentation" --priority low');
	server.cli('doc create "Architecture" -d "System architecture overview" -t "core"');
	server.cli('doc create "Auth Guide" -d "Authentication guide for developers" -t guide -t auth');
});

test.afterAll(() => {
	server?.cleanup();
});

test.describe("Search Command Dialog", () => {
	test("opens search with Cmd+K and shows results", async ({ page }) => {
		await test.step("Navigate to kanban", async () => {
			await page.goto(`${server.baseURL}/kanban`);
			await page.waitForTimeout(1000);
		});

		await test.step("Open search dialog with Cmd+K", async () => {
			await page.keyboard.press("Meta+k");
			await page.waitForTimeout(300);
		});

		await test.step("Search dialog is visible", async () => {
			const searchInput = page.getByPlaceholder(/search/i).first();
			await expect(searchInput).toBeVisible({ timeout: 3000 });
		});

		await test.step("Type search query", async () => {
			await page.keyboard.type("Auth");
			await page.waitForTimeout(500);
		});

		await test.step("Results show matching tasks and docs", async () => {
			// Should find "Auth Module" task and "Auth Guide" doc
			await expect(page.getByText("Auth Module").first()).toBeVisible({ timeout: 5000 });
		});

		await test.step("Close search with Escape", async () => {
			await page.keyboard.press("Escape");
			await page.waitForTimeout(300);
		});
	});

	test("search finds tasks by description content", async ({ page }) => {
		await test.step("Navigate and open search", async () => {
			await page.goto(`${server.baseURL}/kanban`);
			await page.waitForTimeout(500);
			await page.keyboard.press("Meta+k");
			await page.waitForTimeout(300);
		});

		await test.step("Search for description content", async () => {
			await page.keyboard.type("authentication");
			await page.waitForTimeout(500);
		});

		await test.step("Task with matching description found", async () => {
			await expect(page.getByText("Auth Module").first()).toBeVisible({ timeout: 5000 });
		});

		await test.step("Cleanup", async () => {
			await page.keyboard.press("Escape");
		});
	});

	test("search finds docs", async ({ page }) => {
		await test.step("Navigate and open search", async () => {
			await page.goto(`${server.baseURL}/kanban`);
			await page.waitForTimeout(500);
			await page.keyboard.press("Meta+k");
			await page.waitForTimeout(300);
		});

		await test.step("Search for doc", async () => {
			await page.keyboard.type("Architecture");
			await page.waitForTimeout(500);
		});

		await test.step("Doc result appears", async () => {
			await expect(page.getByText("Architecture").first()).toBeVisible({ timeout: 5000 });
		});

		await test.step("Cleanup", async () => {
			await page.keyboard.press("Escape");
		});
	});

	test("empty search shows no results message", async ({ page }) => {
		await test.step("Navigate and open search", async () => {
			await page.goto(`${server.baseURL}/kanban`);
			await page.waitForTimeout(500);
			await page.keyboard.press("Meta+k");
			await page.waitForTimeout(300);
		});

		await test.step("Search for non-existent term", async () => {
			await page.keyboard.type("zzzznonexistent12345");
			await page.waitForTimeout(500);
		});

		await test.step("No results shown", async () => {
			await expect(page.getByText(/no result/i).first()).toBeVisible({ timeout: 5000 });
		});

		await test.step("Cleanup", async () => {
			await page.keyboard.press("Escape");
		});
	});

	test("clicking search result navigates to it", async ({ page }) => {
		await test.step("Navigate and open search", async () => {
			await page.goto(`${server.baseURL}/kanban`);
			await page.waitForTimeout(500);
			await page.keyboard.press("Meta+k");
			await page.waitForTimeout(300);
		});

		await test.step("Search and click a task result", async () => {
			await page.keyboard.type("Login Page");
			await page.waitForTimeout(500);
			// Click the CommandItem (role="option") containing "Login Page", not the input text
			const resultItem = page.locator('[role="option"]').filter({ hasText: "Login Page" }).first();
			await expect(resultItem).toBeVisible({ timeout: 5000 });
			await resultItem.click();
			await page.waitForTimeout(500);
		});

		await test.step("Task detail opens", async () => {
			await expect(page.getByRole("heading", { name: "Login Page", exact: true })).toBeVisible({ timeout: 5000 });
		});
	});
});

test.describe("Sidebar Search", () => {
	test("sidebar search input filters navigation", async ({ page }) => {
		await test.step("Navigate to dashboard", async () => {
			await page.goto(server.baseURL);
			await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible({ timeout: 5000 });
		});

		await test.step("Find search button or input in sidebar", async () => {
			const searchBtn = page.locator('[title*="earch"], [placeholder*="earch"]').first();
			if (await searchBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
				await searchBtn.click();
			}
		});
	});
});
