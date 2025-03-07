import { describe, expect, it } from 'vitest';
import { call, list } from '../src/postgres';

import { Call, DefineConfig, DefineSecret, StandardDecode } from './blaxel';

let silent = true;

describe('Postgres', async () => {
	DefineSecret('password', 'postgres');
	DefineConfig('connectionString', 'postgresql://postgres:postgres@localhost:5432/postgres');

	it('listTools', async () => {
		const { tools } = await list();
		expect(tools).toBeDefined();
		expect(tools.length).toBeGreaterThan(0);
		for (let tool of tools) {
			expect(tool.name).toBeDefined();
			expect(tool.description).toBeDefined();
			expect(tool.inputSchema).toBeDefined();
		}
		let listFunc = ['query'];
		// asert it's inside the list
		expect(listFunc.every((func) => tools.some((tool) => tool.name === func))).toBe(true);
	});

	it('query', async () => {
		let response = await Call(call, 'query', {
			sql: 'SELECT id,name,email FROM users',
		});
		let value = StandardDecode(response);
		expect(value.length).toBeGreaterThan(0);
		expect(value[0].id).toBeDefined();
		expect(value[0].name).toBeDefined();
		expect(value[0].email).toBeDefined();
	});

	it('list_tables', async () => {
		let response = await Call(call, 'list_tables', {});
		let value = StandardDecode(response);
		expect(value.length).toBeGreaterThan(0);
		expect(value[0]).toBeDefined();
	});
});
