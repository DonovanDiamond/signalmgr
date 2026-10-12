<script lang="ts">
	import ThemeToggle from './ThemeToggle.svelte';
	import ChangePasswordDialog from './ChangePasswordDialog.svelte';
	import { session } from '$lib/session.svelte';
	import type { GatewayInfo } from '$lib/types';

	let { gateway, online }: { gateway: GatewayInfo | null; online: boolean } = $props();

	let open = $state(false);
	let changing = $state(false);
	let root = $state<HTMLDivElement>();

	const user = $derived(session.user);
	const roleText = $derived(user?.role === 'admin' ? 'Administrator' : 'User');
	// The role adds nothing for an ordinary user; say it only for the elevated one.
	const detail = $derived(
		[user?.role === 'admin' ? roleText : '', user?.source === 'ad' ? 'Active Directory' : ''].filter(Boolean).join(', ')
	);

	function outside(e: MouseEvent) {
		if (open && !root?.contains(e.target as Node)) open = false;
	}
	function key(e: KeyboardEvent) {
		if (open && e.key === 'Escape') open = false;
	}
</script>

<svelte:window onclick={outside} onkeydown={key} />

{#if user}
	<div class="relative min-w-0" bind:this={root}>
		<button
			type="button"
			class="xbtn max-w-full"
			aria-haspopup="menu"
			aria-expanded={open}
			data-testid="account-menu-button"
			onclick={() => (open = !open)}
		>
			<span class="lamp {online ? 'lamp-on' : 'lamp-bad'}" style="width:10px;height:10px" aria-hidden="true"></span>
			<span class="truncate" data-testid="user-name">{user.name}</span>
			<svg aria-hidden="true" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" class="flex-none"><path d="m6 9 6 6 6-6" /></svg>
		</button>

		{#if open}
			<div class="menu-panel" role="menu" data-testid="account-menu">
				<div class="grid gap-1 p-3" style="border-bottom:1px solid var(--color-base-300)">
					<span class="font-bold break-words">{user.name}</span>
					<span class="narrow muted text-sm break-words">{user.username}{detail ? `, ${detail}` : ''}</span>
				</div>
				<div class="grid gap-1 p-3 text-sm" style="border-bottom:1px solid var(--color-base-300)" data-testid="gateway-status">
					<span class="inline-flex items-center gap-2">
						<span class="lamp {online ? 'lamp-on' : 'lamp-bad'}" style="width:12px;height:12px" aria-hidden="true"></span>
						{online ? 'Gateway online' : 'Gateway unreachable'}
					</span>
					{#if online && gateway}
						<span class="narrow muted" data-testid="gateway-version">API {gateway.version}, {gateway.mode} mode</span>
					{/if}
				</div>
				<div class="grid gap-1 p-2">
					{#if session.isAdmin}
						<a href="/users" class="menu-item" role="menuitem" data-testid="menu-users" onclick={() => (open = false)}>Users</a>
					{/if}
					{#if user.source === 'local'}
						<button type="button" class="menu-item" role="menuitem" data-testid="menu-change-password" onclick={() => { open = false; changing = true; }}>Change my password</button>
					{/if}
					<ThemeToggle class="menu-item" />
					<button type="button" class="menu-item" role="menuitem" data-testid="sign-out" onclick={() => session.logout()}>Sign out</button>
				</div>
			</div>
		{/if}
	</div>

	{#if changing}
		<ChangePasswordDialog onclose={() => (changing = false)} ondone={() => (changing = false)} />
	{/if}
{/if}

<style>
	.menu-panel {
		position: absolute;
		right: 0;
		top: calc(100% + 6px);
		z-index: 30;
		width: 280px;
		max-width: calc(100vw - 2rem);
		background: var(--color-base-100);
		border: 1px solid var(--color-base-300);
		border-radius: 4px;
		box-shadow: 0 6px 20px rgb(0 0 0 / 0.18);
	}
	:global(.menu-item) {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 8px 10px;
		border: 0;
		border-radius: 4px;
		background: transparent;
		color: var(--color-base-content);
		font: 700 14px var(--font-sans);
		text-align: left;
		text-decoration: none;
		cursor: pointer;
	}
	:global(.menu-item:hover) {
		background: var(--color-base-200);
		color: var(--color-base-content);
	}
</style>
