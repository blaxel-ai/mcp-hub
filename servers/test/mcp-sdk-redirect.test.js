"use strict";

// Regression test for @modelcontextprotocol/sdk 1.32 (GHSA-6prh-2h8m-c8cw): the HTTP client
// transport must not re-send custom headers or the request body to another origin when an MCP
// server answers with a cross-origin 307. Both servers listen on loopback; a different port is a
// different origin. The client goes through the SDK's own StreamableHTTPClientTransport.
const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const { Client } = require("@modelcontextprotocol/sdk/client/index.js");
const { StreamableHTTPClientTransport } = require("@modelcontextprotocol/sdk/client/streamableHttp.js");

async function listen(handler) {
	const server = http.createServer(handler);
	await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
	return server;
}

test("a cross-origin 307 does not forward the API key or the request body", async () => {
	const received = [];
	const target = await listen((req, res) => {
		let body = "";
		req.on("data", (chunk) => (body += chunk));
		req.on("end", () => {
			received.push({ key: req.headers["x-api-key"], body });
			res.writeHead(500);
			res.end();
		});
	});
	const origin = await listen((req, res) => {
		req.resume();
		res.writeHead(307, { location: `http://127.0.0.1:${target.address().port}/mcp` });
		res.end();
	});
	const transport = new StreamableHTTPClientTransport(new URL(`http://127.0.0.1:${origin.address().port}/mcp`), {
		requestInit: { headers: { "X-API-Key": "secret" } },
	});
	const client = new Client({ name: "redirect-check", version: "0.0.1" });
	try {
		await client.connect(transport).catch(() => {});
	} finally {
		await client.close().catch(() => {});
		target.close();
		origin.close();
	}
	for (const request of received) {
		assert.equal(request.key, undefined, "the API key reached the other origin");
		assert.ok(!request.body.includes("initialize"), "the request body reached the other origin");
	}
});
