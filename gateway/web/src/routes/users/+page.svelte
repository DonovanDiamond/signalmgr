<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { apiFetch, ApiError, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { session } from '$lib/session.svelte';
	import { toastError } from '$lib/toast.svelte';
	import type { GatewayInfo, ManagedUser } from '$lib/types';
	import Topbar from '$lib/components/Topbar.svelte';
	import CreateUserDialog from '$lib/components/CreateUserDialog.svelte';
	import ResetPasswordDialog from '$lib/components/ResetPasswordDialog.svelte';

	type Dialog = { kind: 'create' } | { kind: 'reset'; username: string };

	let phase = $state<'loading' | 'ready' | 'failed'>('loading');
	let gateway = $state<GatewayInfo | null>(null);
	let unreachable = $state(false);
	let users = $state<ManagedUser[]>([]);
	let dialog = $state<Dialog | null>(null);

	// Only admins belong here; the server enforces it too.
	$effect(() => {
		if (session.user && !session.isAdmin) goto('/');
	});

	async function fetchUsers() {
		users = await apiFetch<ManagedUser[]>('/api/users');
	}

	async function load() {
		phase = 'loading';
		try {
			await fetchUsers();
			phase = 'ready';
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) return;
			phase = 'failed';
			toastError(err instanceof Error ? err.message : 'Could not load users.');
		}
	}

	async function toggle(u: ManagedUser) {
		const ok = await act(
			() => apiFetch(`/api/users/${seg(u.username)}`, { method: 'PATCH', body: JSON.stringify({ disabled: !u.disabled }) }),
			u.disabled ? `${u.username} enabled.` : `${u.username} disabled.`
		);
		if (ok) await fetchUsers();
	}

	function done() {
		dialog = null;
		fetchUsers().catch(() => {});
	}

	onMount(() => {
		if (!session.isAdmin) return;
		load();
		apiFetch<GatewayInfo>('/api/gateway').then(
			(g) => (gateway = g),
			() => (unreachable = true)
		);
	});
</script>

<Topbar {gateway} {unreachable} />

{#if session.isAdmin}
	<main class="mx-auto max-w-[1120px] px-4 pb-12 pt-8 sm:px-6" data-testid="users-page">
		<div class="mb-4 flex flex-wrap items-end justify-between gap-4">
			<div>
				<h1 class="m-0 text-2xl font-bold">Users</h1>
				<p class="muted mt-1 text-[15px]">People who can sign in to this gateway. <a href="/">Back to numbers</a></p>
			</div>
			<button type="button" class="xbtn xbtn-solid xbtn-lg" data-testid="action-create-user" onclick={() => (dialog = { kind: 'create' })}>
				<svg aria-hidden="true" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M12 5v14M5 12h14" /></svg>
				Create a user
			</button>
		</div>

		{#if phase === 'loading'}
			<section class="ledger p-6" aria-busy="true" data-testid="users-loading"><span role="status">Loading users…</span></section>
		{:else if phase === 'failed'}
			<section class="ledger p-8 text-center" data-testid="users-failed">
				<span class="lamp lamp-bad mb-3" aria-hidden="true"></span>
				<h2 class="text-xl font-bold">Users could not be loaded</h2>
				<button type="button" class="xbtn xbtn-solid xbtn-lg mt-5" data-testid="retry" onclick={load}>Try again</button>
			</section>
		{:else}
			<section class="ledger" aria-label="Users" data-testid="users-ledger">
				{#each users as u (u.username)}
					{@const self = u.username === session.user?.username}
					<div class="ledger-item">
						<div class="urow" data-testid="user-row" data-username={u.username} data-disabled={u.disabled}>
							<div class="min-w-0">
								<div class="text-xl font-bold break-words">{u.name}{#if self}<span class="narrow muted ml-1 text-[15px]">(you)</span>{/if}</div>
								<div class="narrow muted text-[15px] break-words">{u.username}</div>
							</div>
							<div>
								{u.role === 'admin' ? 'Administrator' : 'User'}
								<div class="narrow muted text-sm">{u.source === 'ad' ? 'Active Directory, role from groups' : 'Local account'}</div>
							</div>
							<div class="inline-flex items-center gap-2" data-testid="user-state">
								<span class="lamp {u.disabled ? 'lamp-off' : 'lamp-on'}" aria-hidden="true"></span>
								{u.disabled ? 'Disabled' : 'Active'}
							</div>
							<div class="urow-actions">
								{#if u.source === 'ad'}
									<span class="narrow muted text-sm">Managed in the directory</span>
								{:else}
									<button type="button" class="xbtn" data-testid="action-reset-password" onclick={() => (dialog = { kind: 'reset', username: u.username })}>Reset password</button>
									{#if self}
										<span class="narrow muted text-sm">You cannot disable yourself</span>
									{:else}
										<button type="button" class="xbtn {u.disabled ? '' : 'xbtn-danger'}" data-testid="action-toggle-user" onclick={() => toggle(u)}>{u.disabled ? 'Enable' : 'Disable'}</button>
									{/if}
								{/if}
							</div>
						</div>
					</div>
				{/each}
			</section>
		{/if}
	</main>

	{#if dialog?.kind === 'create'}
		<CreateUserDialog onclose={() => (dialog = null)} ondone={done} />
	{:else if dialog?.kind === 'reset'}
		<ResetPasswordDialog username={dialog.username} onclose={() => (dialog = null)} ondone={() => (dialog = null)} />
	{/if}
{/if}

<style>
	.urow {
		display: grid;
		grid-template-columns: minmax(0, 1.5fr) minmax(0, 1.2fr) minmax(0, 0.7fr) minmax(0, 1.4fr);
		align-items: center;
		gap: 12px 24px;
		padding: 16px 24px;
	}
	.urow-actions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: flex-end;
		gap: 8px;
	}
	@media (max-width: 720px) {
		.urow {
			grid-template-columns: minmax(0, 1fr);
			padding: 16px;
		}
		.urow-actions {
			justify-content: flex-start;
		}
	}
</style>
