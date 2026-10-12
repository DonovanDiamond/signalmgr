<script lang="ts">
	import '../app.css';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { session } from '$lib/session.svelte';
	import Toasts from '$lib/components/Toasts.svelte';

	let { children } = $props();

	const isLogin = $derived(page.url.pathname === '/login');

	// Every route but /login needs a session; a signed-in visitor has no reason to be on /login.
	$effect(() => {
		if (!session.user && !isLogin) {
			goto('/login');
		} else if (session.user && isLogin) {
			goto('/');
		}
	});
</script>

<svelte:head><title>Signal Gateway</title></svelte:head>

{#if session.user || isLogin}
	{@render children()}
{/if}
<Toasts />
