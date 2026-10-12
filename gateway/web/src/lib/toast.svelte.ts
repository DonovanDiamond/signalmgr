export interface Toast {
	id: number;
	kind: 'error' | 'success';
	text: string;
}

const SUCCESS_MS = 5000;

let nextId = 1;
export const toasts = $state<Toast[]>([]);
const timers = new Map<number, ReturnType<typeof setTimeout>>();

export function dismiss(id: number) {
	clearTimeout(timers.get(id));
	timers.delete(id);
	const i = toasts.findIndex((t) => t.id === id);
	if (i >= 0) toasts.splice(i, 1);
}

function arm(id: number) {
	clearTimeout(timers.get(id));
	timers.set(id, setTimeout(() => dismiss(id), SUCCESS_MS));
}

function push(kind: Toast['kind'], text: string) {
	// The same message again is not a new fact: keep one toast, and give a success a fresh countdown.
	const same = toasts.find((t) => t.kind === kind && t.text === text);
	if (same) {
		if (kind === 'success') arm(same.id);
		return;
	}
	const id = nextId++;
	toasts.push({ id, kind, text });
	// Errors stay until dismissed so they cannot be missed.
	if (kind === 'success') arm(id);
}

export const toastError = (text: string) => push('error', text);
export const toastOk = (text: string) => push('success', text);
