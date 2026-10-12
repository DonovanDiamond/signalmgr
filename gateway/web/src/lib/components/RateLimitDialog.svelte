<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { formatNumber } from '$lib/format';

	let { number, onclose, ondone }: { number: string; onclose: () => void; ondone: () => void } = $props();

	let challengeToken = $state('');
	let captcha = $state('');
	let busy = $state(false);

	const problem = $derived(
		!challengeToken.trim()
			? 'Enter the challenge token from the rate-limit error.'
			: !captcha.trim().startsWith('signalcaptcha://')
				? 'Paste the whole captcha token, which starts with signalcaptcha://'
				: ''
	);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const ok = await act(
			() =>
				apiFetch(`/api/numbers/${seg(number)}/rate-limit`, {
					method: 'POST',
					body: JSON.stringify({ challengeToken: challengeToken.trim(), captcha: captcha.trim() })
				}),
			'Challenge submitted.'
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Rate-limit challenge for {formatNumber(number)}" testid="dialog-rate-limit" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<p class="muted text-sm">
			Solve the captcha at
			<a href="https://signalcaptchas.org/challenge/generate.html" target="_blank" rel="noopener noreferrer">signalcaptchas.org</a>
			and paste the token below with the challenge token Signal returned.
		</p>
		<div>
			<label class="field-label" for="rl-challenge">Challenge token</label>
			<input id="rl-challenge" class="input w-full" autocomplete="off" bind:value={challengeToken} required />
		</div>
		<div>
			<label class="field-label" for="rl-captcha">Captcha token</label>
			<textarea id="rl-captcha" class="textarea w-full" rows="3" placeholder="signalcaptcha://…" bind:value={captcha} required></textarea>
			{#if problem}<p class="field-hint" role="status">{problem}</p>{/if}
		</div>
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>{busy ? 'Submitting…' : 'Submit challenge'}</button>
		</div>
	</form>
</Modal>
