import { goto } from '$app/navigation';
import { session } from './session.svelte';

export class ApiError extends Error {
	status: number;
	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

/** Every call goes through here, so an expired session is handled once: clear it, go to login. */
async function request(path: string, init: RequestInit = {}): Promise<Response> {
	const headers = new Headers(init.headers);
	if (session.token) {
		headers.set('Authorization', `Bearer ${session.token}`);
	}
	if (init.body && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}

	const res = await fetch(path, { ...init, headers });

	// A failed sign-in is also a 401, but there is no session to expire and the server's reason must reach the form.
	if (res.status === 401 && path !== '/api/auth/login') {
		session.clear();
		goto('/login');
		throw new ApiError(401, await errorMessage(res));
	}
	if (!res.ok) {
		throw new ApiError(res.status, await errorMessage(res));
	}
	return res;
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
	const res = await request(path, init);
	if (res.status === 204) {
		return undefined as T;
	}
	return (await res.json()) as T;
}

/** For authenticated binary responses (the link QR): an <img src> cannot send the bearer token. */
export async function apiBlob(path: string): Promise<Blob> {
	return (await request(path)).blob();
}

/** Path segment for a phone number: the leading "+" must be escaped. */
export function seg(value: string | number): string {
	return encodeURIComponent(String(value));
}

async function errorMessage(res: Response): Promise<string> {
	try {
		const body = await res.json();
		if (body && typeof body.error === 'string') return body.error;
	} catch {
		// non-JSON body; fall through to a generic message
	}
	return res.statusText || `request failed with status ${res.status}`;
}
