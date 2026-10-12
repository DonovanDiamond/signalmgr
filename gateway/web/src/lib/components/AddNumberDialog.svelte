<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, ApiError } from '$lib/api';
	import { toastOk } from '$lib/toast.svelte';
	import { E164 } from '$lib/format';

	let { onclose, ondone }: { onclose: () => void; ondone: (number: string) => void } = $props();

	let number = $state('');
	let label = $state('');
	let busy = $state(false);
	let failure = $state('');

	const problem = $derived(
		E164.test(number.trim()) ? '' : 'Enter the number in international format, with + and the country code.'
	);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const n = number.trim();
		failure = '';
		try {
			await apiFetch('/api/numbers', { method: 'POST', body: JSON.stringify({ number: n, label: label.trim() }) });
		} catch (err) {
			// A 401 has already sent the user to login.
			if (!(err instanceof ApiError && err.status === 401)) {
				failure =
					err instanceof ApiError && err.status === 409
						? 'This number is already on the gateway.'
						: err instanceof Error
							? err.message
							: 'Something went wrong.';
			}
			busy = false;
			return;
		}
		busy = false;
		toastOk('Number added.');
		ondone(n);
	}
</script>

<Modal title="Add a number" testid="dialog-add-number" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<div>
			<label class="field-label" for="add-number">Phone number</label>
			<input id="add-number" class="input w-full" type="tel" placeholder="e.g. +27 82 555 0143" bind:value={number} required />
		</div>
		<div>
			<label class="field-label" for="add-label">Label (optional)</label>
			<input id="add-label" class="input w-full" type="text" placeholder="e.g. Reception" bind:value={label} />
		</div>
		{#if problem}<p class="field-hint" role="status">{problem}</p>{:else if failure}<p class="field-hint" role="alert" data-testid="add-number-error">{failure}</p>{/if}
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>{busy ? 'Adding…' : 'Add number'}</button>
		</div>
	</form>
</Modal>
