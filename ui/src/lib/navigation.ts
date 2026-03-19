import { router } from "../router";

export function navigateTo(to: string, options?: { replace?: boolean }) {
	return router.navigate({
		to,
		replace: options?.replace,
	});
}
