<script lang="ts">
	import { formatNumber, ageMs, ago, RECENT_MS } from '$lib/format';
	import { session } from '$lib/session.svelte';
	import type { Device, SignalNumber } from '$lib/types';

	let {
		item,
		expanded,
		ontoggle,
		onaction
	}: {
		item: SignalNumber;
		expanded: boolean;
		ontoggle: () => void;
		onaction: (kind: 'register' | 'verify' | 'add-device' | 'rename' | 'pin' | 'unregister' | 'rate-limit' | 'remove-device', device?: Device) => void;
	} = $props();

	const stateInfo = $derived(
		{
			registered: { lamp: 'lamp-on', text: 'Registered' },
			awaiting_code: { lamp: 'lamp-wait', text: 'Awaiting code' },
			not_registered: { lamp: 'lamp-off', text: 'Not registered' }
		}[item.state]
	);
	const count = $derived(item.devices.length);
	const recent = (d: Device) => {
		const a = ageMs(d.lastSeen);
		return d.primary || (a !== null && a < RECENT_MS);
	};
	const id = $derived(`devices-${item.number.replace(/\D/g, '')}`);
</script>

<div class="ledger-item" data-testid="number-row" data-number={item.number} data-state={item.state}>
	<div class="row">
		<div>
			<div class="row-num" style:color={item.state === 'not_registered' ? 'var(--muted)' : undefined}>{formatNumber(item.number)}</div>
			<div class="narrow muted mt-0.5 flex items-center gap-3 text-[15px]">
				{#if item.label}<span data-testid="row-label">{item.label}</span>{/if}
				{#if session.isAdmin}
					<button type="button" class="xlink" style="font-weight:500" data-testid="action-rename" onclick={() => onaction('rename')}>{item.label ? 'Rename' : 'Add a label'}</button>
				{/if}
			</div>
		</div>
		<div class="inline-flex items-center gap-2" data-testid="row-state">
			<span class="lamp {stateInfo.lamp}" aria-hidden="true"></span>
			<span>{stateInfo.text}</span>
		</div>
		<div class="inline-flex items-center gap-1.5 text-sm">
			{#if item.state === 'registered' && count > 0}
				{#each item.devices as d (d.id)}
					<span class="lamp {recent(d) ? 'lamp-on' : 'lamp-hollow'}" aria-hidden="true"></span>
				{/each}
				<span class="ml-1" data-testid="device-count">{count} {count === 1 ? 'device' : 'devices'}</span>
			{:else}
				<span class="muted">{item.state === 'awaiting_code' ? 'No devices yet' : 'No devices'}</span>
			{/if}
		</div>
		<div class="row-actions">
			{#if item.state === 'registered'}
				<button type="button" class="xbtn" aria-expanded={expanded} aria-controls={id} data-testid="expand-devices" onclick={ontoggle}>
					{expanded ? 'Hide devices' : 'Show devices'}
				</button>
			{:else if session.isAdmin && item.state === 'awaiting_code'}
				<button type="button" class="xbtn xbtn-solid" data-testid="action-verify" onclick={() => onaction('verify')}>Enter code</button>
				<button type="button" class="xbtn" data-testid="action-register" onclick={() => onaction('register')}>Register again</button>
			{:else if session.isAdmin}
				<button type="button" class="xbtn" data-testid="action-register" onclick={() => onaction('register')}>Register number</button>
			{/if}
		</div>
	</div>

	{#if expanded && item.state === 'registered'}
		<div class="sub" {id} data-testid="device-ledger">
			<div class="sub-head">
				<span class="narrow text-[15px]">Linked devices</span>
				{#if session.isAdmin}
					<button type="button" class="xlink" data-testid="action-add-device" onclick={() => onaction('add-device')}>Add a device to this number</button>
				{/if}
			</div>
			{#each item.devices as d (d.id)}
				<div class="sub-row" data-testid="device-row" data-device-id={d.id}>
					<div class="inline-flex items-center gap-2.5">
						<span class="lamp {recent(d) ? 'lamp-on' : 'lamp-hollow'}" aria-hidden="true"></span>
						<span class="font-bold">{d.name}</span>
					</div>
					<div class="text-sm tabular-nums">ID {d.id}</div>
					<div class="sub-meta muted text-sm">{d.primary ? 'Primary device' : ago(d.lastSeen)}</div>
					<div class="muted text-sm">
						{#if d.primary}
							Cannot be removed
						{:else if session.isAdmin}
							<button type="button" class="xbtn xbtn-danger" style="padding:6px 12px" aria-label="Remove {d.name}" data-testid="action-remove-device" onclick={() => onaction('remove-device', d)}>Remove</button>
						{/if}
					</div>
				</div>
			{:else}
				<div class="muted p-4 text-sm">No devices linked.</div>
			{/each}
			{#if session.isAdmin}
				<div class="sub-foot" style="border-top:1px solid var(--color-base-300)">
					<button type="button" class="xlink" data-testid="action-pin" onclick={() => onaction('pin')}>Set or remove PIN</button>
					<button type="button" class="xlink" data-testid="action-rate-limit" onclick={() => onaction('rate-limit')}>Rate-limit challenge</button>
					<button type="button" class="xlink" style="color:var(--danger)" data-testid="action-unregister" onclick={() => onaction('unregister')}>Unregister</button>
				</div>
			{/if}
		</div>
	{/if}
</div>
