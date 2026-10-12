<script lang="ts">
	import Modal from './Modal.svelte';
	import { apiFetch, seg } from '$lib/api';
	import { act } from '$lib/act';
	import { formatNumber } from '$lib/format';

	let { number, onclose, ondone }: { number: string; onclose: () => void; ondone: () => void } = $props();

	let captcha = $state('');
	let method = $state<'sms' | 'voice'>('sms');
	let busy = $state(false);

	const problem = $derived(
		captcha.trim().startsWith('signalcaptcha://')
			? ''
			: 'Paste the whole token, which starts with signalcaptcha://'
	);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const ok = await act(
			() =>
				apiFetch(`/api/numbers/${seg(number)}/register`, {
					method: 'POST',
					body: JSON.stringify({ captcha: captcha.trim(), voice: method === 'voice' })
				}),
			'Verification code requested.'
		);
		busy = false;
		if (ok) ondone();
	}
</script>

<Modal title="Register {formatNumber(number)}" testid="dialog-register" {onclose}>
	<form onsubmit={submit} class="grid gap-4">
		<p class="muted text-sm">
			Solve the captcha at
			<a href="https://signalcaptchas.org/registration/generate.html" target="_blank" rel="noopener noreferrer">signalcaptchas.org</a>,
			then copy the link behind “Open Signal” and paste it here.
		</p>
		<div>
			<label class="field-label" for="reg-captcha">Captcha token</label>
			<textarea id="reg-captcha" class="textarea w-full" rows="3" placeholder="signalcaptcha://…" bind:value={captcha} required></textarea>
			{#if captcha && problem}<p class="field-hint" role="status">{problem}</p>{/if}
		</div>
		<fieldset>
			<legend class="field-label">Send the code by</legend>
			<label class="mr-6 inline-flex items-center gap-2"><input type="radio" class="radio radio-sm" name="method" value="sms" bind:group={method} /> SMS</label>
			<label class="inline-flex items-center gap-2"><input type="radio" class="radio radio-sm" name="method" value="voice" bind:group={method} /> Voice call</label>
		</fieldset>
		<div class="flex justify-end gap-2">
			<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
			<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>{busy ? 'Requesting…' : 'Request code'}</button>
		</div>
	</form>
</Modal>
