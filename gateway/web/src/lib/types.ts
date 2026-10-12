export type Role = 'admin' | 'user';

export interface User {
	username: string;
	name: string;
	role: Role;
}

export type NumberState = 'registered' | 'awaiting_code' | 'not_registered';

export interface Device {
	id: number;
	name: string;
	created: string;
	lastSeen: string;
	primary: boolean;
}

export interface SignalNumber {
	number: string;
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
