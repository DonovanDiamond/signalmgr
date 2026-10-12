export type Role = 'admin' | 'user';

export type Source = 'local' | 'ad';

export interface User {
	username: string;
	name: string;
	role: Role;
	source: Source;
}

export interface ManagedUser extends User {
	disabled: boolean;
}

/** Mirrors config.MinPasswordLength on the server. */
export const MIN_PASSWORD_LENGTH = 12;

export type NumberState = 'registered' | 'awaiting_code' | 'not_registered';

export interface Device {
	id: number;
	name: string;
	created: string;
	lastSeen: string;
	primary: boolean;
}

export interface SignalNumber {
	/** Account identifier used in URLs: the number, or the ACI for a numberless account. */
	id: string;
	/** Empty for an account that has no phone number. */
	number: string;
	/** Empty when the gateway does not know the account's ACI. */
	aci: string;
	label: string;
	state: NumberState;
	devices: Device[];
}

export interface GatewayInfo {
	online: boolean;
	version: string;
	mode: string;
	installationName: string;
}
