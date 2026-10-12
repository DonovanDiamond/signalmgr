<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { formatNumber } from '$lib/format';

	let {
		number,
		current,
		title,
		onclose,
		ondone
	}: { number: string; current: string; title: string; onclose: () => void; ondone: () => void } = $props();

	// svelte-ignore state_referenced_locally -- the dialog is mounted per use, so the starting value is read once
	let label = $state(current);
	let busy = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const ok = await act(
			() => apiFetch(`/api/numbers/${seg(number)}`, { method: 'PATCH', body: JSON.stringify({ label: label.trim() }) }),
			'Label saved.'
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal {title} testid="dialog-rename" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<div>
			<label class="field-label" for="rename-label">Label for {formatNumber(number)}</label>
			<input id="rename-label" class="input w-full" type="text" placeholder="e.g. Reception" bind:value={label} />
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Skip</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy}>{busy ? 'Saving…' : 'Save label'}</button>
		</div>
	</form>
</Modal>
