import { ApiError } from './api';
import { toastError, toastOk } from './toast.svelte';

/** Run a mutation; show the API's error text on failure. Returns whether it succeeded. */
export async function act(fn: () => Promise<unknown>, success?: string): Promise<boolean> {
	try {
		await fn();
		if (success) toastOk(success);
		return true;
	} catch (err) {
		// A 401 has already sent the user to login.
		if (!(err instanceof ApiError && err.status === 401)) {
			toastError(err instanceof Error ? err.message : 'Something went wrong.');
		}
		return false;
	}
}
