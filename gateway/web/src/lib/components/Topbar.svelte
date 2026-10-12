<script lang="ts">
	import ThemeToggle from './ThemeToggle.svelte';
	import { session } from '$lib/session.svelte';
	import type { GatewayInfo } from '$lib/types';

	let { gateway, unreachable }: { gateway: GatewayInfo | null; unreachable: boolean } = $props();

	const online = $derived(!!gateway?.online && !unreachable);
</script>

<header class="topbar flex items-center justify-between gap-4 border-b px-8 py-3.5" style="background:var(--color-base-100); border-color:var(--color-base-300)" data-testid="topbar">
	<span class="text-xl font-bold" data-testid="installation-name">{gateway?.installationName || 'Signal Gateway'}</span>
	<div class="flex flex-wrap items-center gap-x-7 gap-y-3 text-sm">
		<span class="inline-flex items-center gap-2" data-testid="gateway-status">
			<span class="lamp {online ? 'lamp-on' : 'lamp-bad'}" style="width:12px;height:12px" aria-hidden="true"></span>
			{online ? 'Gateway online' : 'Gateway unreachable'}
		</span>
		{#if online && gateway}
			<span class="narrow muted" data-testid="gateway-version">API {gateway.version}, {gateway.mode} mode</span>
		{/if}
		{#if session.user}
			<span class="inline-flex items-center gap-2" data-testid="user-menu">
				<svg aria-hidden="true" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="8" r="4" /><path d="M4 21c0-4 4-6 8-6s8 2 8 6" /></svg>
				{session.user.name}, {session.user.role}
			</span>
		{/if}
		<ThemeToggle />
		<button type="button" class="xbtn" data-testid="sign-out" onclick={() => session.logout()}>Sign out</button>
	</div>
</header>
