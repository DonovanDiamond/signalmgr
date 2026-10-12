// Tokens live in memory plus sessionStorage (never localStorage, never logged).
import { goto } from '$app/navigation';
import { apiFetch } from './api';
import type { User } from './types';

const STORAGE_KEY = 'signalgateway.session';

interface StoredSession {
	token: string;
	refreshToken: string;
	user: User;
}

interface AuthResponse {
	token: string;
	refreshToken: string;
	user: User;
}

function readStored(): StoredSession | null {
	if (typeof sessionStorage === 'undefined') return null;
	const raw = sessionStorage.getItem(STORAGE_KEY);
	if (!raw) return null;
	try {
		return JSON.parse(raw) as StoredSession;
	} catch {
		return null;
	}
}

function createSession() {
	const stored = readStored();
	let token = $state<string | null>(stored?.token ?? null);
	let refreshToken = $state<string | null>(stored?.refreshToken ?? null);
	let user = $state<User | null>(stored?.user ?? null);
	let inflight: Promise<boolean> | null = null;

	function persist() {
		if (typeof sessionStorage === 'undefined') return;
		if (token && refreshToken && user) {
			sessionStorage.setItem(STORAGE_KEY, JSON.stringify({ token, refreshToken, user }));
		} else {
			sessionStorage.removeItem(STORAGE_KEY);
		}
	}

	async function login(username: string, password: string) {
		store(
			await apiFetch<AuthResponse>('/api/auth/login', {
				method: 'POST',
				body: JSON.stringify({ username, password })
			})
		);
	}

	function store(res: AuthResponse) {
		token = res.token;
		refreshToken = res.refreshToken;
		user = res.user;
		persist();
	}

	/** Swap the refresh token for a new pair. Concurrent callers share one request, since refresh tokens rotate. */
	function refresh(): Promise<boolean> {
		if (!refreshToken) return Promise.resolve(false);
		inflight ??= (async () => {
			try {
				const res = await fetch('/api/auth/refresh', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ refreshToken })
				});
				if (!res.ok) return false;
				store((await res.json()) as AuthResponse);
				return true;
			} catch {
				return false;
			} finally {
				inflight = null;
			}
		})();
		return inflight;
	}

	function clear() {
		token = null;
		refreshToken = null;
		user = null;
		persist();
	}

	/** Tell the server, but never let a failed call trap the user in a session. */
	async function logout() {
		const bearer = token;
		clear();
		goto('/login');
		if (!bearer) return;
		try {
			await fetch('/api/auth/logout', {
				method: 'POST',
				headers: { Authorization: `Bearer ${bearer}` }
			});
		} catch {
			// ignored by design
		}
	}

	return {
		get token() {
			return token;
		},
		get user() {
			return user;
		},
		get isAdmin() {
			return user?.role === 'admin';
		},
		login,
		refresh,
		clear,
		logout
	};
}

export const session = createSession();
