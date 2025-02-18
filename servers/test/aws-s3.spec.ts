import { describe, expect, it } from 'vitest';
import { call, list } from '../src/aws-s3';

import { Call, DefineConfig, DefineSecret, StandardDecode } from './baxel';

let silent = true;

describe('AWS S3', async () => {
	// @ts-ignore
	const accessKeyId = import.meta.env.AWS_ACCESS_KEY_ID;
	// @ts-ignore
	const secretAccessKey = import.meta.env.AWS_SECRET_ACCESS_KEY;
	// @ts-ignore
	const sessionToken = import.meta.env.AWS_SESSION_TOKEN;
	// @ts-ignore
	const region = import.meta.env.AWS_REGION;
	DefineSecret('accessKeyId', accessKeyId);
	DefineSecret('secretAccessKey', secretAccessKey);
	DefineSecret('sessionToken', sessionToken);
	DefineConfig('region', region);

	it('listTools', async () => {
		const { tools } = await list();
		expect(tools).toBeDefined();
		expect(tools.length).toBeGreaterThan(0);
		for (let tool of tools) {
			expect(tool.name).toBeDefined();
			expect(tool.description).toBeDefined();
			expect(tool.inputSchema).toBeDefined();
		}
		let listFunc = ['list_files', 'retrieve_file'];
		// asert it's inside the list
		expect(listFunc.every((func) => tools.some((tool) => tool.name === func))).toBe(true);
	});

	it('list_files', async () => {
		let response = await Call(call, 'list_files', {
			bucket: 'cli-baxel-test',
			prefix: '',
		});
		let value = StandardDecode(response);
		expect(value.length).toBeGreaterThan(0);
	});

	it('upload_file', async () => {
		let response = await Call(call, 'upload_file', {
			bucket: 'cli-baxel-test',
			key: 'hello.txt',
			data: 'Hello, world!',
		});
		let value = StandardDecode(response);
		expect(value).toBeDefined();
	});

	it('retrieve_file', async () => {
		let response = await Call(call, 'retrieve_file', {
			bucket: 'cli-baxel-test',
			key: 'hello.txt',
		});
		let value = StandardDecode(response);
		expect(value).toBeDefined();
	});
});
