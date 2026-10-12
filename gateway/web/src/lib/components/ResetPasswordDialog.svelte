<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { MIN_PASSWORD_LENGTH } from '$lib/types';

	let { username, onclose, ondone }: { username: string; onclose: () => void; ondone: () => void } = $props();

	let password = $state('');
	let busy = $state(false);

	const problem = $derived(
		password && password.length < MIN_PASSWORD_LENGTH ? `The password needs at least ${MIN_PASSWORD_LENGTH} characters.` : ''
	);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const ok = await act(
			() => apiFetch(`/api/users/${seg(username)}/password`, { method: 'PUT', body: JSON.stringify({ password }) }),
			`Password reset for ${username}.`
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Reset password for {username}" testid="dialog-reset-password" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<div>
			<label class="field-label" for="rp-password">New password</label>
			<input id="rp-password" class="input w-full" type="password" autocomplete="new-password" bind:value={password} required />
			<p class="field-hint">At least {MIN_PASSWORD_LENGTH} characters.</p>
		</div>
		{#if problem}<p class="field-hint" role="status">{problem}</p>{/if}
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !password || !!problem}>{busy ? 'Saving…' : 'Reset password'}</button>
		</div>
	</form>
</Modal>
