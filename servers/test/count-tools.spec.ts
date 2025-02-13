import { describe, it } from 'vitest';
import { mcpServers } from '../src/servers';

let silent = true;

describe('Count', async () => {
	it('toolkit', async () => {
		const tools = Object.keys(mcpServers);
		console.log(`Toolkits: ${tools.join(', ')}`);
		console.log(`Count: ${tools.length}`);
	});
	it('tools', async () => {
		const listing = await Promise.all(
			Object.keys(mcpServers).map(async (server) => {
				const list = await mcpServers[server].list();
				return list.tools;
			}),
		);
		const tools = listing.flat();
		console.log(`Tools: ${tools.map((tool) => tool.name).join(', ')}`);
		console.log(`Count: ${tools.length}`);
	});
});
