<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch } from '$lib/api';
	import { act } from '$lib/act';
	import { MIN_PASSWORD_LENGTH } from '$lib/types';

	let { onclose, ondone }: { onclose: () => void; ondone: () => void } = $props();

	let current = $state('');
	let next = $state('');
	let confirm = $state('');
	let busy = $state(false);

	const problem = $derived(
		!current
			? 'Enter your current password.'
			: next.length < MIN_PASSWORD_LENGTH
				? `The new password needs at least ${MIN_PASSWORD_LENGTH} characters.`
				: confirm !== next
					? 'The two new passwords do not match.'
					: ''
	);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const ok = await act(
			() => apiFetch('/api/auth/password', { method: 'POST', body: JSON.stringify({ current, new: next }) }),
			'Password changed.'
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Change my password" testid="dialog-change-password" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<div>
			<label class="field-label" for="pw-current">Current password</label>
			<input id="pw-current" class="input w-full" type="password" autocomplete="current-password" bind:value={current} required />
		</div>
		<div>
			<label class="field-label" for="pw-new">New password</label>
			<input id="pw-new" class="input w-full" type="password" autocomplete="new-password" bind:value={next} required />
			<p class="field-hint">At least {MIN_PASSWORD_LENGTH} characters.</p>
		</div>
		<div>
			<label class="field-label" for="pw-confirm">Repeat the new password</label>
			<input id="pw-confirm" class="input w-full" type="password" autocomplete="new-password" bind:value={confirm} required />
		</div>
		{#if problem}<p class="field-hint" role="status">{problem}</p>{/if}
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>{busy ? 'Saving…' : 'Change password'}</button>
		</div>
	</form>
</Modal>
