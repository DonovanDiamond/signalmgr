/** "+27825550143" -> "+27 82 555 0143"; other numbers are shown as stored. */
export function formatNumber(n: string): string {
	const m = /^\+27(\d{2})(\d{3})(\d{4})$/.exec(n);
	return m ? `+27 ${m[1]} ${m[2]} ${m[3]}` : n;
}

/** The first 8 characters of an ACI, enough to tell accounts apart. */
export const shortAci = (aci: string) => aci.slice(0, 8);

/** Name for an account in dialog titles: its number, else its label, else a short ACI. */
export function accountName(a: { number: string; label: string; aci: string }): string {
	return a.number ? formatNumber(a.number) : a.label || `account ${shortAci(a.aci)}`;
}

export const E164 = /^\+[1-9]\d{6,14}$/;

export const RECENT_MS = 24 * 3600 * 1000;

export function ageMs(iso: string, now = Date.now()): number | null {
	const t = Date.parse(iso);
	return Number.isNaN(t) ? null : Math.max(0, now - t);
}

export function ago(iso: string): string {
	const ms = ageMs(iso);
	if (ms === null) return 'Last seen unknown';
	const min = Math.floor(ms / 60000);
	if (min < 1) return 'Last seen just now';
	if (min < 60) return `Last seen ${min} min ago`;
	const h = Math.floor(min / 60);
	if (h < 48) return `Last seen ${h} ${h === 1 ? 'hour' : 'hours'} ago`;
	return `Last seen ${Math.floor(h / 24)} days ago`;
}
