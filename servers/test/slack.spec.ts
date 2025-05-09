import { describe, expect, it } from 'vitest';
import { call, list } from '../src/slack';

import { Call, DefineConfig, DefineSecret, StandardDecode } from './blaxel';

let silent = true;

describe('Slack', async () => {
	// @ts-ignore
	const apiKey = import.meta.env.SLACK_BOT_TOKEN;
	DefineSecret('bot_token', apiKey);
	DefineConfig('team_id', 'blaxel-ai');

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
			'slack_list_channels',
			'slack_post_message',
			'slack_reply_to_thread',
			'slack_add_reaction',
			'slack_get_channel_history',
			'slack_get_thread_replies',
			'slack_get_users',
			'slack_get_user_profile',
		];
		// asert it's inside the list
		expect(listFunc.every((func) => tools.some((tool) => tool.name === func))).toBe(true);
	});

	it('slack_list_channels', async () => {
		let response = await Call(call, 'slack_list_channels', {
			limit: 2,
		});
		let value = StandardDecode(response);
		expect(value.ok).toBe(true);
		expect(value.channels.length).toBeGreaterThan(0);
		const channel = value.channels[0];
		expect(channel.id).toBeDefined();
		expect(channel.name).toBeDefined();
		expect(channel.is_private).toBeDefined();
		expect(channel.is_private).toBe(false);
		expect(channel.is_private).toBe(false);
	});

	let ts: string = '';
	it('slack_post_message', async () => {
		let response = await Call(call, 'slack_post_message', {
			channel_id: 'C08AVFMPGLB',
			text: 'Hello, world!',
		});
		let value = StandardDecode(response);
		expect(value.ok).toBe(true);
		expect(value.channel).toBeDefined();
		expect(value.ts).toBeDefined();
		ts = value.ts;
		expect(value.message.user).toBeDefined();
		expect(value.message.text).toBe('Hello, world!');
	});

	it('slack_reply_to_thread', async () => {
		let response = await Call(call, 'slack_reply_to_thread', {
			channel_id: 'C08AVFMPGLB',
			thread_ts: ts,
			text: 'Hello, world response!',
		});
		let valueResponse = StandardDecode(response);
		expect(valueResponse.ok).toBe(true);
	});
});
