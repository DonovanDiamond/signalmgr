<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import type { Device } from '$lib/types';

	let { id, name, device, onclose, ondone }: { id: string; name: string; device: Device; onclose: () => void; ondone: () => void } = $props();

	let busy = $state(false);

	async function remove() {
		busy = true;
		const ok = await act(
			() => apiFetch(`/api/numbers/${seg(id)}/devices/${seg(device.id)}`, { method: 'DELETE' }),
			'Device removed.'
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Remove device" testid="dialog-remove-device" {onclose}>
	<p class="mb-4">
		Remove <strong>{device.name}</strong> (ID {device.id}) from {name}? That device will be signed out of Signal.
	</p>
	<div class="flex justify-end gap-2">
		<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
		<button type="button" class="xbtn xbtn-danger-solid" data-testid="submit" disabled={busy} onclick={remove}>{busy ? 'Removing…' : 'Remove device'}</button>
	</div>
</Modal>
