<script lang="ts">
	import { theme } from '$lib/theme.svelte';
</script>

<button
	type="button"
	class="xbtn"
	data-testid="theme-toggle"
	aria-label={theme.current === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
	onclick={() => theme.toggle()}
>
	<svg aria-hidden="true" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
		{#if theme.current === 'dark'}
			<circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
		{:else}
			<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" />
		{/if}
	</svg>
	{theme.current === 'dark' ? 'Light' : 'Dark'}
</button>
