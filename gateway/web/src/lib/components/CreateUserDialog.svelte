<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch } from '$lib/api';
	import { act } from '$lib/act';
	import { MIN_PASSWORD_LENGTH, type Role } from '$lib/types';

	let { onclose, ondone }: { onclose: () => void; ondone: () => void } = $props();

	let username = $state('');
	let name = $state('');
	let role = $state<Role>('user');
	let password = $state('');
	let busy = $state(false);

	const problem = $derived(
		!username.trim()
			? 'Enter a username.'
			: !name.trim()
				? 'Enter the person’s name.'
				: password.length < MIN_PASSWORD_LENGTH
					? `The password needs at least ${MIN_PASSWORD_LENGTH} characters.`
					: ''
	);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const u = username.trim();
		const ok = await act(
			() => apiFetch('/api/users', { method: 'POST', body: JSON.stringify({ username: u, name: name.trim(), role, password }) }),
			`User ${u} created.`
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Create a user" testid="dialog-create-user" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<div>
			<label class="field-label" for="cu-username">Username</label>
			<input id="cu-username" class="input w-full" type="text" autocomplete="off" bind:value={username} required />
		</div>
		<div>
			<label class="field-label" for="cu-name">Display name</label>
			<input id="cu-name" class="input w-full" type="text" autocomplete="off" bind:value={name} required />
		</div>
		<div>
			<label class="field-label" for="cu-role">Role</label>
			<select id="cu-role" class="select w-full" bind:value={role}>
				<option value="user">User: can see numbers</option>
				<option value="admin">Administrator: can change everything</option>
			</select>
		</div>
		<div>
			<label class="field-label" for="cu-password">Initial password</label>
			<input id="cu-password" class="input w-full" type="password" autocomplete="new-password" bind:value={password} required />
			<p class="field-hint">At least {MIN_PASSWORD_LENGTH} characters. Tell the person to change it after signing in.</p>
		</div>
		{#if problem}<p class="field-hint" role="status">{problem}</p>{/if}
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>{busy ? 'Creating…' : 'Create user'}</button>
		</div>
	</form>
</Modal>
