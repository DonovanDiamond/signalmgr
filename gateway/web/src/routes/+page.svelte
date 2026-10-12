<script lang="ts">
	import { onMount } from 'svelte';
	import { formatNumber } from '$lib/format';
	import { apiFetch, ApiError } from '$lib/api';
	import { session } from '$lib/session.svelte';
	import { toastError } from '$lib/toast.svelte';
	import type { Device, GatewayInfo, SignalNumber } from '$lib/types';
	import Topbar from '$lib/components/Topbar.svelte';
	import NumberRow from '$lib/components/NumberRow.svelte';
	import AddNumberDialog from '$lib/components/AddNumberDialog.svelte';
	import RegisterDialog from '$lib/components/RegisterDialog.svelte';
	import VerifyDialog from '$lib/components/VerifyDialog.svelte';
	import LinkGatewayDialog from '$lib/components/LinkGatewayDialog.svelte';
	import AddDeviceDialog from '$lib/components/AddDeviceDialog.svelte';
	import RenameDialog from '$lib/components/RenameDialog.svelte';
	import RemoveDeviceDialog from '$lib/components/RemoveDeviceDialog.svelte';
	import PinDialog from '$lib/components/PinDialog.svelte';
	import UnregisterDialog from '$lib/components/UnregisterDialog.svelte';
	import RateLimitDialog from '$lib/components/RateLimitDialog.svelte';

	type Dialog =
		| { kind: 'add' }
		| { kind: 'link-gateway' }
		| { kind: 'label'; number: string }
		| { kind: 'register' | 'verify' | 'add-device' | 'rename' | 'pin' | 'unregister' | 'rate-limit'; number: string }
		| { kind: 'remove-device'; number: string; device: Device };

	let phase = $state<'loading' | 'ready' | 'unreachable'>('loading');
	let problem = $state('');
	let gateway = $state<GatewayInfo | null>(null);
	let numbers = $state<SignalNumber[]>([]);
	let expanded = $state<Record<string, boolean>>({});
	let dialog = $state<Dialog | null>(null);

	async function fetchNumbers(): Promise<SignalNumber[]> {
		numbers = await apiFetch<SignalNumber[]>('/api/numbers');
		return numbers;
	}

	async function load() {
		phase = 'loading';
		problem = '';
		try {
			gateway = await apiFetch<GatewayInfo>('/api/gateway');
			if (!gateway.online) {
				phase = 'unreachable';
				return;
			}
			await fetchNumbers();
			phase = 'ready';
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) return;
			phase = 'unreachable';
			problem = err instanceof ApiError && err.status !== 502 ? err.message : '';
			if (!(err instanceof ApiError)) problem = 'The gateway could not be reached from this browser.';
		}
	}

	// After a mutation: refresh the list without flashing the loading state.
	async function refresh() {
		try {
			await fetchNumbers();
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) return;
			toastError(err instanceof Error ? err.message : 'Could not refresh the list.');
		}
	}

	function done(next?: Dialog) {
		dialog = next ?? null;
		refresh();
	}

	onMount(load);

	const awaiting = $derived(numbers.filter((n) => n.state === 'awaiting_code'));
	const summary = $derived.by(() => {
		const total = `${numbers.length} ${numbers.length === 1 ? 'number' : 'numbers'}.`;
		if (awaiting.length === 0) return total;
		if (awaiting.length === 1) return `${total} ${awaiting[0].label || formatNumber(awaiting[0].number)} is waiting for its code.`;
		return `${total} ${awaiting.length} are waiting for their codes.`;
	});

	type RowKind = Exclude<Dialog['kind'], 'add' | 'link-gateway' | 'label'>;
	function onaction(n: SignalNumber, kind: RowKind, device?: Device) {
		if (kind === 'remove-device' && device) dialog = { kind, number: n.number, device };
		else if (kind !== 'remove-device') dialog = { kind, number: n.number };
	}

	const labelOf = (number: string) => numbers.find((n) => n.number === number)?.label ?? '';
</script>

<Topbar {gateway} unreachable={phase === 'unreachable'} />

<main class="mx-auto max-w-[1120px] px-6 pb-12 pt-8" data-testid="ledger-page">
	<div class="ledger-head mb-4 flex items-end justify-between gap-4">
		<div>
			<h1 class="m-0 text-2xl font-bold">Numbers on this gateway</h1>
			{#if phase === 'ready' && numbers.length > 0}<p class="muted mt-1 text-[15px]" data-testid="summary">{summary}</p>{/if}
		</div>
		{#if session.isAdmin && phase === 'ready' && numbers.length > 0}
			<div class="head-actions flex flex-wrap gap-2">
				<button type="button" class="xbtn xbtn-lg" data-testid="action-link-gateway" onclick={() => (dialog = { kind: 'link-gateway' })}>Link this gateway to a phone</button>
				<button type="button" class="xbtn xbtn-solid xbtn-lg" data-testid="action-add-number" onclick={() => (dialog = { kind: 'add' })}>
					<svg aria-hidden="true" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M12 5v14M5 12h14" /></svg>
					Add a number
				</button>
			</div>
		{/if}
	</div>

	{#if phase === 'loading'}
		<section class="ledger" aria-label="Line ledger" aria-busy="true" data-testid="ledger-loading">
			{#each [0, 1, 2] as i (i)}
				<div class="ledger-item">
					<div class="row">
						<div class="grid gap-2"><div class="skeleton-bar" style="height:34px;width:70%"></div><div class="skeleton-bar" style="height:15px;width:35%"></div></div>
						<div class="skeleton-bar" style="height:18px;width:60%"></div>
						<div class="skeleton-bar" style="height:18px;width:50%"></div>
						<div class="skeleton-bar" style="height:36px;width:110px"></div>
					</div>
				</div>
			{/each}
			<span class="sr-only" role="status">Loading numbers…</span>
		</section>
	{:else if phase === 'unreachable'}
		<section class="ledger p-8 text-center" aria-label="Gateway unreachable" data-testid="ledger-unreachable">
			<span class="lamp lamp-bad mb-3" aria-hidden="true"></span>
			<h2 class="text-xl font-bold">The Signal gateway is not answering</h2>
			<p class="muted mx-auto mt-2 max-w-prose">
				{problem || 'The signal-cli service this app talks to is offline or unreachable. Numbers cannot be listed or changed until it is back.'}
			</p>
			<button type="button" class="xbtn xbtn-solid xbtn-lg mt-5" data-testid="retry" onclick={load}>Try again</button>
		</section>
	{:else if numbers.length === 0}
		<section class="ledger p-10 text-center" aria-label="Line ledger" data-testid="ledger-empty">
			<span class="lamp lamp-off mb-3" aria-hidden="true"></span>
			<h2 class="text-xl font-bold">No numbers yet</h2>
			<p class="muted mx-auto mt-2 max-w-prose">
				{session.isAdmin
					? 'Add the first phone number this gateway should send from. You will register it with Signal in the next steps.'
					: 'An administrator has not added any numbers to this gateway yet.'}
			</p>
			{#if session.isAdmin}
				<div class="mt-5 flex flex-wrap justify-center gap-2">
					<button type="button" class="xbtn xbtn-lg" data-testid="action-link-gateway-empty" onclick={() => (dialog = { kind: 'link-gateway' })}>Link this gateway to a phone</button>
					<button type="button" class="xbtn xbtn-solid xbtn-lg" data-testid="action-add-number-empty" onclick={() => (dialog = { kind: 'add' })}>Add a number</button>
				</div>
			{/if}
		</section>
	{:else}
		<section class="ledger" aria-label="Line ledger" data-testid="ledger">
			{#each numbers as n (n.number)}
				<NumberRow
					item={n}
					expanded={!!expanded[n.number]}
					ontoggle={() => (expanded[n.number] = !expanded[n.number])}
					onaction={(kind, device) => onaction(n, kind, device)}
				/>
			{/each}
		</section>
		<p class="muted mt-4 text-sm">
			Lamp key: green is registered, amber is waiting for a code, grey is not registered. Filled device dots were seen in the last day; a hollow dot has not been seen for over a day.
		</p>
	{/if}
</main>

{#if dialog?.kind === 'add'}
	<AddNumberDialog onclose={() => (dialog = null)} ondone={(number) => done({ kind: 'register', number })} />
{:else if dialog?.kind === 'register'}
	<RegisterDialog number={dialog.number} onclose={() => (dialog = null)} ondone={() => done({ kind: 'verify', number: (dialog as { number: string }).number })} />
{:else if dialog?.kind === 'verify'}
	<VerifyDialog number={dialog.number} onclose={() => (dialog = null)} ondone={() => done()} />
{:else if dialog?.kind === 'link-gateway'}
	<LinkGatewayDialog {numbers} load={fetchNumbers} onclose={() => (dialog = null)} ondone={(number) => done({ kind: 'label', number })} />
{:else if dialog?.kind === 'label'}
	<RenameDialog number={dialog.number} current={labelOf(dialog.number)} title="Label the linked number" onclose={() => (dialog = null)} ondone={() => done()} />
{:else if dialog?.kind === 'rename'}
	<RenameDialog number={dialog.number} current={labelOf(dialog.number)} title="Rename" onclose={() => (dialog = null)} ondone={() => done()} />
{:else if dialog?.kind === 'add-device'}
	<AddDeviceDialog number={dialog.number} onclose={() => (dialog = null)} ondone={() => done()} />
{:else if dialog?.kind === 'remove-device'}
	<RemoveDeviceDialog number={dialog.number} device={dialog.device} onclose={() => (dialog = null)} ondone={() => done()} />
{:else if dialog?.kind === 'pin'}
	<PinDialog number={dialog.number} onclose={() => (dialog = null)} ondone={() => done()} />
{:else if dialog?.kind === 'unregister'}
	<UnregisterDialog number={dialog.number} onclose={() => (dialog = null)} ondone={() => done()} />
{:else if dialog?.kind === 'rate-limit'}
	<RateLimitDialog number={dialog.number} onclose={() => (dialog = null)} ondone={() => done()} />
{/if}
