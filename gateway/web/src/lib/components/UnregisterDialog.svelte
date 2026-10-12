<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { accountName, formatNumber, shortAci } from '$lib/format';
	import type { SignalNumber } from '$lib/types';

	let { item, onclose, ondone }: { item: SignalNumber; onclose: () => void; ondone: () => void } = $props();

	let deleteAccount = $state(false);
	let deleteLocalData = $state(false);
	let typed = $state('');
	let busy = $state(false);

	const name = $derived(accountName(item));
	// Numbered: the number. Numberless: its label, else the start of its ACI.
	const expected = $derived(item.number || item.label || shortAci(item.aci));
	const squash = (s: string) => s.replace(/\s/g, '');
	const matches = $derived(squash(typed) === squash(expected));

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const ok = await act(
			() =>
				apiFetch(`/api/numbers/${seg(item.id)}/unregister`, {
					method: 'POST',
					body: JSON.stringify({ deleteAccount, deleteLocalData })
				}),
			'Number unregistered.'
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Unregister {name}" testid="dialog-unregister" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<label class="flex items-start gap-2"><input type="checkbox" class="checkbox checkbox-sm mt-1" name="deleteAccount" bind:checked={deleteAccount} />
			<span>Also delete the Signal account (cannot be undone; contacts will lose this {item.number ? 'number' : 'account'})</span></label>
		<label class="flex items-start gap-2"><input type="checkbox" class="checkbox checkbox-sm mt-1" name="deleteLocalData" bind:checked={deleteLocalData} />
			<span>Also delete the data stored on this gateway</span></label>
		<div>
			<label class="field-label" for="unreg-confirm">Type {formatNumber(expected)} to confirm</label>
			<input id="unreg-confirm" class="input w-full" autocomplete="off" placeholder={formatNumber(expected)} bind:value={typed} />
			{#if typed && !matches}<p class="field-hint" role="status">That does not match {formatNumber(expected)}.</p>{/if}
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-danger-solid" data-testid="submit" disabled={busy || !matches}>{busy ? 'Unregistering…' : 'Unregister'}</button>
		</div>
	</form>
</Modal>
