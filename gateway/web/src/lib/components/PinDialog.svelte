<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { formatNumber } from '$lib/format';

	let { number, onclose, ondone }: { number: string; onclose: () => void; ondone: () => void } = $props();

	let pin = $state('');
	let busy = $state(false);

	const problem = $derived(pin.length >= 4 ? '' : 'A PIN is at least 4 characters.');

	async function set(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const ok = await act(
			() => apiFetch(`/api/numbers/${seg(number)}/pin`, { method: 'PUT', body: JSON.stringify({ pin }) }),
			'PIN set.'
		);
		busy = false;
		if (ok) ondone();
	}

	async function remove() {
		busy = true;
		const ok = await act(() => apiFetch(`/api/numbers/${seg(number)}/pin`, { method: 'DELETE' }), 'PIN removed.');
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="PIN for {formatNumber(number)}" testid="dialog-pin" {onclose}>
	<form onsubmit={set} class="grid gap-4">
		<div>
			<label class="field-label" for="pin-new">New PIN</label>
			<input id="pin-new" class="input w-full" type="password" autocomplete="new-password" bind:value={pin} required />
			{#if pin && problem}<p class="field-hint" role="status">{problem}</p>{/if}
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>Set PIN</button>
		</div>
	</form>
	<div class="mt-5 flex items-center justify-between gap-3 border-t pt-4" style="border-color:var(--color-base-300)">
		<span class="muted text-sm">Remove the PIN this number currently has.</span>
		<button type="button" class="xbtn xbtn-danger" data-testid="remove-pin" disabled={busy} onclick={remove}>Remove PIN</button>
	</div>
</Modal>
