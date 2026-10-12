<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { formatNumber } from '$lib/format';

	let { number, onclose, ondone }: { number: string; onclose: () => void; ondone: () => void } = $props();

	let deleteAccount = $state(false);
	let deleteLocalData = $state(false);
	let typed = $state('');
	let busy = $state(false);

	const matches = $derived(typed.replace(/\s/g, '') === number);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const ok = await act(
			() =>
				apiFetch(`/api/numbers/${seg(number)}/unregister`, {
					method: 'POST',
					body: JSON.stringify({ deleteAccount, deleteLocalData })
				}),
			'Number unregistered.'
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Unregister {formatNumber(number)}" testid="dialog-unregister" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<label class="flex items-start gap-2"><input type="checkbox" class="checkbox checkbox-sm mt-1" name="deleteAccount" bind:checked={deleteAccount} />
			<span>Also delete the Signal account (cannot be undone; contacts will lose this number)</span></label>
		<label class="flex items-start gap-2"><input type="checkbox" class="checkbox checkbox-sm mt-1" name="deleteLocalData" bind:checked={deleteLocalData} />
			<span>Also delete the data stored on this gateway</span></label>
		<div>
			<label class="field-label" for="unreg-confirm">Type {formatNumber(number)} to confirm</label>
			<input id="unreg-confirm" class="input w-full" autocomplete="off" placeholder={formatNumber(number)} bind:value={typed} />
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-danger-solid" data-testid="submit" disabled={busy || !matches}>{busy ? 'Unregistering…' : 'Unregister'}</button>
		</div>
	</form>
</Modal>
