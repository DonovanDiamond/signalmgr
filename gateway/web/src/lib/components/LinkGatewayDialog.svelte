<script lang="ts">
	import { onDestroy } from 'svelte';
	import Modal from './Modal.svelte';
	import { apiBlob, apiFetch } from '$lib/api';
	import { act } from '$lib/act';
	import { toastError } from '$lib/toast.svelte';
	import type { SignalNumber } from '$lib/types';

	let {
		numbers,
		load,
		onclose,
		ondone
	}: {
		numbers: SignalNumber[];
		load: () => Promise<SignalNumber[]>;
		onclose: () => void;
		ondone: (id: string) => void;
	} = $props();

	let deviceName = $state('');
	let qrUrl = $state('');
	let linkUri = $state('');
	let waiting = $state(false);
	let busy = $state(false);
	let timer: ReturnType<typeof setInterval> | undefined;

	// Fixed when the QR is issued, so an account that existed before is never mistaken for the new one.
	let known = new Set<string>();

	const problem = $derived(deviceName.trim() ? '' : 'Give this gateway a name.');

	function stop() {
		if (timer) clearInterval(timer);
		timer = undefined;
	}

	async function start(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const q = encodeURIComponent(deviceName.trim());
		const ok = await act(async () => {
			const [blob, uri] = await Promise.all([
				apiBlob(`/api/link/qr?deviceName=${q}`),
				apiFetch<{ uri: string }>(`/api/link/uri?deviceName=${q}`)
			]);
			qrUrl = URL.createObjectURL(blob);
			linkUri = uri.uri;
		});
		busy = false;
		if (!ok) return;
		known = new Set(numbers.map((n) => n.id));
		waiting = true;
		timer = setInterval(poll, 3000);
	}

	async function poll() {
		try {
			const added = (await load()).find((n) => !known.has(n.id));
			if (added) {
				stop();
				ondone(added.id);
			}
		} catch {
			// a transient failure must not end the wait; the next tick retries
		}
	}

	async function copy() {
		try {
			await navigator.clipboard.writeText(linkUri);
		} catch {
			toastError('Could not copy. Select the link text and copy it by hand.');
		}
	}

	onDestroy(() => {
		stop();
		if (qrUrl) URL.revokeObjectURL(qrUrl);
	});
</script>

<Modal title="Link this gateway to a phone" testid="dialog-link-gateway" onclose={() => { stop(); onclose(); }}>
	{#if !waiting}
		<form onsubmit={start} class="grid gap-4">
			<div>
				<label class="field-label" for="link-name">Name for this gateway</label>
				<input id="link-name" class="input w-full" placeholder="e.g. Front desk gateway" bind:value={deviceName} required />
				<p class="field-hint">Shown in Signal under Linked devices on the phone.</p>
			</div>
			<div class="flex justify-end gap-2">
				<button type="button" class="xbtn" onclick={onclose}>Cancel</button>
				<button type="submit" class="xbtn xbtn-solid" data-testid="submit" disabled={busy || !!problem}>{busy ? 'Preparing…' : 'Show QR code'}</button>
			</div>
		</form>
	{:else}
		<div class="grid justify-items-center gap-3">
			<p class="muted text-center text-sm">In Signal on the phone: Settings, Linked devices, then scan this code.</p>
			<img src={qrUrl} alt="QR code to link {deviceName}" class="h-56 w-56 bg-white p-2" data-testid="link-qr" />
			<p class="flex items-center gap-2" role="status" data-testid="link-waiting">
				<span class="lamp lamp-wait"></span> Waiting for scan
			</p>
			<button type="button" class="xbtn" data-testid="copy-link" onclick={copy}>Copy link</button>
		</div>
		<div class="mt-4 flex justify-end">
			<button type="button" class="xbtn" onclick={() => { stop(); onclose(); }}>Close</button>
		</div>
	{/if}
</Modal>
