<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch } from '$lib/api';
	import { act } from '$lib/act';
	import { E164 } from '$lib/format';

	let { onclose, ondone }: { onclose: () => void; ondone: (number: string) => void } = $props();

	let number = $state('');
	let label = $state('');
	let busy = $state(false);

	const problem = $derived(
		E164.test(number.trim()) ? '' : 'Enter the number in international format, for example +27825550143.'
	);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const n = number.trim();
		const ok = await act(
			() => apiFetch('/api/numbers', { method: 'POST', body: JSON.stringify({ number: n, label: label.trim() }) }),
			'Number added.'
		);
		busy = false;
		if (ok) ondone(n);
	}
</script>

<Modal title="Add a number" testid="dialog-add-number" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<div>
			<label class="field-label" for="add-number">Phone number</label>
			<input id="add-number" class="input w-full" type="tel" placeholder="+27825550143" bind:value={number} required />
		</div>
		<div>
			<label class="field-label" for="add-label">Label (optional)</label>
			<input id="add-label" class="input w-full" type="text" placeholder="Reception" bind:value={label} />
		</div>
		{#if number && problem}<p class="field-hint" role="status">{problem}</p>{/if}
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>{busy ? 'Adding…' : 'Add number'}</button>
		</div>
	</form>
</Modal>
