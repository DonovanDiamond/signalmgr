// Token lives in memory plus sessionStorage (never localStorage, never logged).
import { goto } from '$app/navigation';
import { apiFetch } from './api';
import type { User } from './types';

const STORAGE_KEY = 'signalgateway.session';

interface StoredSession {
	token: string;
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
	let user = $state<User | null>(stored?.user ?? null);

	function persist() {
		if (typeof sessionStorage === 'undefined') return;
		if (token && user) {
			sessionStorage.setItem(STORAGE_KEY, JSON.stringify({ token, user }));
		} else {
			sessionStorage.removeItem(STORAGE_KEY);
		}
	}

	async function login(username: string, password: string) {
		const res = await apiFetch<{ token: string; user: User }>('/api/auth/login', {
			method: 'POST',
			body: JSON.stringify({ username, password })
		});
		token = res.token;
		user = res.user;
		persist();
	}

	function clear() {
		token = null;
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
		clear,
		logout
	};
}

export const session = createSession();
