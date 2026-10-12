<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { formatNumber } from '$lib/format';

	let { number, onclose, ondone }: { number: string; onclose: () => void; ondone: () => void } = $props();

	let uri = $state('');
	let busy = $state(false);

	const problem = $derived(uri.trim().startsWith('sgnl://linkdevice?') ? '' : 'The link starts with sgnl://linkdevice?');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const ok = await act(
			() => apiFetch(`/api/numbers/${seg(number)}/devices`, { method: 'POST', body: JSON.stringify({ uri: uri.trim() }) }),
			'Device linked.'
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Add a device to {formatNumber(number)}" testid="dialog-add-device" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<div>
			<label class="field-label" for="device-uri">Link from the new device</label>
			<input id="device-uri" class="input w-full" autocomplete="off" spellcheck="false" placeholder="sgnl://linkdevice?uuid=…" bind:value={uri} required />
			<p class="field-hint">In Signal Desktop or iPad, choose to link this device, then copy the link shown under the QR code.</p>
			{#if uri && problem}<p class="field-hint" role="status">{problem}</p>{/if}
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>{busy ? 'Linking…' : 'Link device'}</button>
		</div>
	</form>
</Modal>
