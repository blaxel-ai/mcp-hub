import type { Context } from 'shared/types';
import { expect } from 'vitest';

const secrets: Record<string, string> = {};
const config: Record<string, string> = {};

export function DefineSecret(key: string, value: string) {
	secrets[key] = value;
}
export function DefineConfig(key: string, value: string) {
	config[key] = value;
}

export async function Call(func: Function, name: string, args: any) {
	let context: Context = {
		debugMode: true,
		spanCtx: {
			traceId: '123',
			spanId: '456',
			traceFlags: 0,
		},
	};
	let request = new Request('http://localhost', {
		method: 'POST',
		body: JSON.stringify({
			name,
			arguments: args,
		}),
	});

	// @ts-ignore
	return await func(context, request, config, secrets, import.meta.env);
}

export function StandardDecode(response: any) {
	if (response.isError) {
		expect(response.content[0].text).toBeDefined();
		let error = JSON.parse(response.content[0].text);
		throw new Error(error.error.message || error.error);
	}
	expect(response.isError, `There is an error : ${response.content[0].text}`).toBe(false);

	expect(response).toBeDefined();
	expect(response.content).toBeDefined();
	expect(response.content.length).toBeGreaterThan(0);
	if (response.content[0].type === 'text') {
		try {
			return JSON.parse(response.content[0].text);
		} catch (error) {
			return response.content[0].text;
		}
	}
	if (response.content[0].type === 'file') {
		return response.content[0];
	}
	throw new Error('Unknown response type');
}
