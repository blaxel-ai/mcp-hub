"use strict";

// Smoke test for the `@modelcontextprotocol/sdk` bump to 1.31.0
// (GHSA-6qxp-vccf-f47h, OAuth credentials bound to their issuer). This
// project uses no OAuth provider, so the test covers what it does use: a real
// client <-> server round-trip (initialize, tools/list, tools/call) over the
// SDK's in-memory transport.
const test = require("node:test");
const assert = require("node:assert/strict");
const { Server } = require("@modelcontextprotocol/sdk/server/index.js");
const { Client } = require("@modelcontextprotocol/sdk/client/index.js");
const { InMemoryTransport } = require("@modelcontextprotocol/sdk/inMemory.js");
const { ListToolsRequestSchema, CallToolRequestSchema } = require("@modelcontextprotocol/sdk/types.js");

test("MCP client and server round-trip tools/list and tools/call", async () => {
	const server = new Server({ name: "smoke", version: "0.0.1" }, { capabilities: { tools: {} } });
	server.setRequestHandler(ListToolsRequestSchema, async () => ({
		tools: [{ name: "echo", description: "echo", inputSchema: { type: "object", properties: { text: { type: "string" } } } }],
	}));
	server.setRequestHandler(CallToolRequestSchema, async (req) => ({
		content: [{ type: "text", text: String(req.params.arguments.text) }],
	}));
	const client = new Client({ name: "smoke-client", version: "0.0.1" });
	const [clientTransport, serverTransport] = InMemoryTransport.createLinkedPair();
	await Promise.all([server.connect(serverTransport), client.connect(clientTransport)]);
	try {
		const { tools } = await client.listTools();
		assert.deepEqual(tools.map((t) => t.name), ["echo"]);
		const result = await client.callTool({ name: "echo", arguments: { text: "hi" } });
		assert.deepEqual(result.content, [{ type: "text", text: "hi" }]);
	} finally {
		await client.close();
	}
});
