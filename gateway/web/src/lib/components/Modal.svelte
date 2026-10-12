<script lang="ts">
	import { onMount, type Snippet } from 'svelte';

	let {
		title,
		testid,
		onclose,
		children
	}: { title: string; testid: string; onclose: () => void; children: Snippet } = $props();

	let el: HTMLDialogElement;
	const titleId = $derived(`${testid}-title`);

	// Parents mount a dialog only while it is wanted, so opening is mounting.
	onMount(() => el.showModal());
</script>

<dialog bind:this={el} class="modal" data-testid={testid} aria-labelledby={titleId} {onclose}>
	<div class="modal-box" style="border-radius:4px; border:1px solid var(--color-base-300)">
		<h2 id={titleId} class="mb-3 text-xl font-bold">{title}</h2>
		{@render children()}
	</div>
	<form method="dialog" class="modal-backdrop"><button aria-label="Close dialog">close</button></form>
</dialog>
