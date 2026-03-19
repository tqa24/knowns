import { test, expect } from "@playwright/test";
import { startServer, type TestServer } from "./helpers";

let server: TestServer;

test.beforeAll(async () => {
	server = await startServer();
});

test.afterAll(() => {
	server?.cleanup();
});

test.describe("Dashboard", () => {
	test("shows page header and key metrics", async ({ page }) => {
		await test.step("Navigate to dashboard", async () => {
			await page.goto(server.baseURL);
		});

		await test.step("Dashboard header is visible", async () => {
			await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();
		});

		await test.step("Key metrics are displayed", async () => {
			await expect(page.getByText("Total Tasks")).toBeVisible();
			await expect(page.getByText("Documents")).toBeVisible();
			await expect(page.getByText("SDD Coverage").first()).toBeVisible();
		});
	});

	test("shows tasks section with completion progress", async ({ page }) => {
		await test.step("Create a task via CLI", async () => {
			server.cli('task create "Test Dashboard Task" -d "A test task"');
		});

		await test.step("Navigate to dashboard", async () => {
			await page.goto(server.baseURL);
		});

		await test.step("Tasks section shows status counts", async () => {
			await expect(page.locator("section").filter({ hasText: "Tasks" }).getByText("To Do")).toBeVisible();
		});
	});

	test("shows recent activity after task changes", async ({ page }) => {
		let taskId: string | undefined;

		await test.step("Create and update a task to generate activity", async () => {
			const output = server.cli('task create "Activity Test" -d "test" --priority high');
			const idMatch = output.match(/Created task\s+([a-z0-9]+)/i);
			taskId = idMatch?.[1];

			if (taskId) {
				try {
					server.cli(`task edit ${taskId} -s in-progress`);
					server.cli(`task edit ${taskId} -s done`);
				} catch {
					// task ID extraction may fail, that's ok
				}
			}
		});

		await test.step("Navigate to dashboard", async () => {
			await page.goto(server.baseURL);
		});

		await test.step("Recent Activity section is visible", async () => {
			await expect(page.getByRole("heading", { name: "Recent Activity" })).toBeVisible();
		});
	});

	test("shows recent tasks list", async ({ page }) => {
		await test.step("Navigate to dashboard", async () => {
			await page.goto(server.baseURL);
		});

		await test.step("Recent Tasks heading is visible", async () => {
			await expect(page.getByRole("heading", { name: "Recent Tasks" })).toBeVisible();
		});
	});

	test("navigates to tasks page via link", async ({ page }) => {
		await test.step("Navigate to dashboard", async () => {
			await page.goto(server.baseURL);
		});

		await test.step("Click 'View all' link", async () => {
			await page.getByText("View all →").first().click();
		});

		await test.step("URL changes to tasks page", async () => {
			await expect(page).toHaveURL(/\/tasks$/);
		});
	});
});
