<script lang="ts">
	import { toasts, dismiss } from '$lib/toast.svelte';

	let box: HTMLDivElement;

	// A popover lives in the top layer, so a toast stays readable above an open modal dialog.
	// Re-showing moves it above a dialog that was opened after the previous toast.
	$effect(() => {
		if (toasts.length > 0) {
			if (box.matches(':popover-open')) box.hidePopover();
			box.showPopover();
		} else if (box.matches(':popover-open')) {
			box.hidePopover();
		}
	});
</script>

<div
	bind:this={box}
	popover="manual"
	class="toast max-w-[92vw]"
	style="inset:auto 1rem 1rem auto; margin:0; padding:0; border:0; background:transparent; overflow:visible"
	data-testid="toasts"
>
	{#each toasts as t (t.id)}
		<div
			class="alert {t.kind === 'error' ? 'alert-error' : 'alert-success'} items-start"
			role={t.kind === 'error' ? 'alert' : 'status'}
			data-testid="toast-{t.kind}"
		>
			<span class="whitespace-normal break-words">{t.text}</span>
			<button type="button" class="xlink" style="color:inherit" aria-label="Dismiss" onclick={() => dismiss(t.id)}>Dismiss</button>
		</div>
	{/each}
</div>
