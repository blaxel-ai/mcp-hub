import { describe, expect, it } from 'vitest';
import { call, list } from '../src/aws-ses';

import { Call, DefineConfig, DefineSecret, StandardDecode } from './beamlit';


let silent = true;

describe('AWS SES', async() => {
	DefineSecret('accessKeyId', '')
	DefineSecret('secretAccessKey', '')
	DefineSecret('sessionToken', '')
	DefineConfig('region', 'us-west-2')

	it('listTools', async () => {
		const {tools} = await list();
		expect(tools).toBeDefined();
		expect(tools.length).toBeGreaterThan(0);
		for(let tool of tools) {
			expect(tool.name).toBeDefined();
			expect(tool.description).toBeDefined();
			expect(tool.inputSchema).toBeDefined();
		}
		let listFunc = [
			'send_email',
		]
		// asert it's inside the list
		expect(listFunc.every((func) => tools.some((tool) => tool.name === func))).toBe(true);
	})

	it('send_email', async () => {
		let response = await Call(call, 'send_email', {
			from: 'no-reply@beamlit.com',
			to: 'cdrappier@beamlit.com',
			subject: 'Automated email from agent',
			body: 'Hello Charles, you are the sender and the receiver',
		})
		StandardDecode(response)
	})
});
