import { describe, expect, it } from 'vitest';
import { call, list } from '../src/github';

import { Call, DefineSecret, StandardDecode } from './baxel';

let silent = true;

describe('Github', async () => {
	// @ts-ignore
	const personalAccessToken = import.meta.env.GITHUB_PERSONAL_ACCESS_TOKEN;
	DefineSecret('personalAccessToken', personalAccessToken);

	it('listTools', async () => {
		const { tools } = await list();
		expect(tools).toBeDefined();
		expect(tools.length).toBeGreaterThan(0);
		for (let tool of tools) {
			expect(tool.name).toBeDefined();
			expect(tool.description).toBeDefined();
			expect(tool.inputSchema).toBeDefined();
		}
		let listFunc = [
			'fork_repository',
			'create_branch',
			'search_repositories',
			'create_repository',
			'get_file_contents',
			'create_or_update_file',
			'push_files',
			'create_issue',
			'create_pull_request',
			'search_code',
			'search_issues',
			'search_users',
			'list_issues',
			'update_issue',
			'add_issue_comment',
			'list_commits',
			'get_issue',
		];
		expect(listFunc.every((func) => tools.some((tool) => tool.name === func))).toBe(true);
	});
	it('list_issues', async () => {
		let response = await Call(call, 'list_issues', { owner: 'baxel', repo: 'controlplane' });
		let value = StandardDecode(response);
		expect(value).toBeDefined();
		expect(value).instanceOf(Array);
	});
	it('search_repositories', async () => {
		let response = await Call(call, 'search_repositories', { query: 'baxel/controlplane' });
		let value = StandardDecode(response);
		expect(value.total_count).toEqual(1);
	});
});
