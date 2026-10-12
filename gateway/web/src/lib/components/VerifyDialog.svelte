<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { formatNumber } from '$lib/format';

	let { number, onclose, ondone }: { number: string; onclose: () => void; ondone: () => void } = $props();

	let code = $state('');
	let pin = $state('');
	let busy = $state(false);

	const problem = $derived(/^\d{6}$/.test(code.trim()) ? '' : 'The code is 6 digits.');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const body: { code: string; pin?: string } = { code: code.trim() };
		if (pin) body.pin = pin;
		const ok = await act(
			() => apiFetch(`/api/numbers/${seg(number)}/verify`, { method: 'POST', body: JSON.stringify(body) }),
			'Number registered.'
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Enter code for {formatNumber(number)}" testid="dialog-verify" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<div>
			<label class="field-label" for="verify-code">6-digit code</label>
			<input id="verify-code" class="input w-full tabular-nums" inputmode="numeric" autocomplete="one-time-code" maxlength="6" placeholder="123456" bind:value={code} required />
			{#if code && problem}<p class="field-hint" role="status">{problem}</p>{/if}
		</div>
		<div>
			<label class="field-label" for="verify-pin">Registration PIN (only if the number has one)</label>
			<input id="verify-pin" class="input w-full" type="password" autocomplete="off" bind:value={pin} />
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>{busy ? 'Verifying…' : 'Verify'}</button>
		</div>
	</form>
</Modal>
