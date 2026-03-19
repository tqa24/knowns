import { defineConfig } from "@playwright/test";

export default defineConfig({
	testDir: "./e2e",
	timeout: 60_000,
	retries: 0,
	workers: 1, // sequential — each test manages its own server
	reporter: [["list"], ["html", { open: "never" }]],
	use: {
		headless: true,
		screenshot: "on",
		trace: "on",
	},
	projects: [
		{
			name: "chromium",
			use: { browserName: "chromium" },
		},
	],
});
