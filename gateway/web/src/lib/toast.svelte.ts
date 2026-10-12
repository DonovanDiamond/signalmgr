export interface Toast {
	id: number;
	kind: 'error' | 'success';
	text: string;
}

let nextId = 1;
export const toasts = $state<Toast[]>([]);

export function dismiss(id: number) {
	const i = toasts.findIndex((t) => t.id === id);
	if (i >= 0) toasts.splice(i, 1);
}

function push(kind: Toast['kind'], text: string) {
	const id = nextId++;
	toasts.push({ id, kind, text });
	setTimeout(() => dismiss(id), kind === 'error' ? 8000 : 4000);
}

export const toastError = (text: string) => push('error', text);
export const toastOk = (text: string) => push('success', text);
