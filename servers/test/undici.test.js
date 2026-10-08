"use strict";

// Smoke test for the `undici` override bump to 6.28.1, pulled in transitively
// via @qdrant/js-client-rest -> undici. servers/src/qdrant constructs
// `new QdrantClient({ url, apiKey })` and calls getCollections(); this test
// does exactly that against a loopback listener, so the request goes through
// the client's own undici-backed transport.
const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const { createRequire } = require("node:module");

test("QdrantClient getCollections() round-trips through the bumped undici", async () => {
	const { QdrantClient } = await import("@qdrant/js-client-rest");
	const seen = {};
	const server = http.createServer((req, res) => {
		seen.url = req.url;
		seen.apiKey = req.headers["api-key"];
		res.setHeader("Content-Type", "application/json");
		res.end(JSON.stringify({ result: { collections: [{ name: "memories" }] }, status: "ok", time: 0 }));
	});
	await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
	try {
		const client = new QdrantClient({
			url: `http://127.0.0.1:${server.address().port}`,
			apiKey: "dummy-key",
			checkCompatibility: false,
		});
		const response = await client.getCollections();
		assert.deepEqual(response.collections, [{ name: "memories" }]);
		assert.equal(seen.url, "/collections");
		assert.equal(seen.apiKey, "dummy-key");
	} finally {
		server.close();
	}
	const undiciVersion = createRequire(require.resolve("@qdrant/js-client-rest"))("undici/package.json").version;
	assert.equal(undiciVersion, "6.28.1");
});
