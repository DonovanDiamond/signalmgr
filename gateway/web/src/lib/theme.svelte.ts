// System preference by default, user override persisted, applied as daisyUI's data-theme.
const STORAGE_KEY = 'signalgateway.theme';
type Theme = 'light' | 'dark';
const NAMES: Record<Theme, string> = { light: 'exchange', dark: 'exchange-dark' };

function systemTheme(): Theme {
	if (typeof window === 'undefined' || !window.matchMedia) return 'light';
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function readStored(): Theme | null {
	if (typeof localStorage === 'undefined') return null;
	const raw = localStorage.getItem(STORAGE_KEY);
	return raw === 'light' || raw === 'dark' ? raw : null;
}

function createTheme() {
	const stored = readStored();
	let current = $state<Theme>(stored ?? systemTheme());
	let overridden = stored !== null;

	function apply() {
		if (typeof document === 'undefined') return;
		document.documentElement.setAttribute('data-theme', NAMES[current]);
	}
	apply();

	if (typeof window !== 'undefined' && window.matchMedia) {
		window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (event) => {
			if (overridden) return;
			current = event.matches ? 'dark' : 'light';
			apply();
		});
	}

	function toggle() {
		current = current === 'dark' ? 'light' : 'dark';
		overridden = true;
		localStorage.setItem(STORAGE_KEY, current);
		apply();
	}

	return {
		get current() {
			return current;
		},
		toggle
	};
}

export const theme = createTheme();
