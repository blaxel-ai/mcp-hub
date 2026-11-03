// Runtime smoke for CI (.github/workflows/mcp-runtime-smoke.yaml): drive a running MCP image with the SDK client and
// fail on anything but the exact answer. client.ts logs tool errors and exits 0, so it can't gate a merge.
//   node src/smoke.mjs http://localhost:1400
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp.js';

const base = process.argv[2] || 'http://localhost:1400';
const fail = (msg) => { console.error(`FAIL ${msg}`); process.exit(1); };

const client = new Client({ name: 'mcp-runtime-smoke', version: '1.0.0' });
await client.connect(new StreamableHTTPClientTransport(new URL('/mcp', base)));
console.log('ok   initialize:', client.getServerVersion()?.name);

const { tools } = await client.listTools();
if (!tools.some((t) => t.name === 'echo')) fail(`echo not in tools: ${tools.map((t) => t.name).join(', ')}`);
console.log('ok   tools/list:', tools.map((t) => t.name).join(', '));

// Several calls at once: the gateway multiplexes them over one stdio child.
const texts = Array.from({ length: 5 }, (_, i) => `ping-${i}-${Math.random().toString(36).slice(2)}`);
const results = await Promise.all(texts.map((text) => client.callTool({ name: 'echo', arguments: { text } })));
results.forEach((r, i) => {
	const got = r?.content?.[0]?.text;
	if (r?.isError || got !== texts[i]) fail(`echo ${texts[i]} answered ${JSON.stringify(r)}`);
});
console.log(`ok   tools/call: ${texts.length} concurrent echo calls answered exactly`);
await client.close();
