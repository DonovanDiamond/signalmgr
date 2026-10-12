<script lang="ts">
	import AccountMenu from './AccountMenu.svelte';
	import type { GatewayInfo } from '$lib/types';

	let { gateway, unreachable }: { gateway: GatewayInfo | null; unreachable: boolean } = $props();

	const online = $derived(!!gateway?.online && !unreachable);
</script>

<!-- Deliberately not the .topbar class: its small-screen rule stacks the bar, and this one stays on a single row. -->
<header class="flex items-center justify-between gap-3 border-b px-4 py-3 sm:px-8" style="background:var(--color-base-100); border-color:var(--color-base-300)" data-testid="topbar">
	<a href="/" class="min-w-0 truncate text-xl font-bold" style="color:inherit; text-decoration:none" data-testid="installation-name">{gateway?.installationName || 'Signal Gateway'}</a>
	<AccountMenu {gateway} {online} />
</header>
