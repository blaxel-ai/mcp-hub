"use strict";

// Smoke test for the `axios` bump to 1.20.0, pulled in transitively via
// twilio -> axios. servers/src/twilio builds `new Twilio(sid, token)` and lets
// it use its default RequestClient, which is an axios instance. This test does
// the same and only steers the request at a loopback listener, so it proves
// the bumped axios still carries a real round-trip (method, basic auth,
// status, JSON body) through the construction path production uses.
const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const { createRequire } = require("node:module");
const { Twilio } = require("twilio");

test("twilio's default axios-based client round-trips through the bumped axios", async () => {
	const seen = {};
	const server = http.createServer((req, res) => {
		seen.method = req.method;
		seen.auth = req.headers.authorization;
		res.setHeader("Content-Type", "application/json");
		res.end(JSON.stringify({ ok: true }));
	});
	await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
	try {
		const client = new Twilio("AC00000000000000000000000000000000", "dummy-token");
		const response = await client.request({
			method: "get",
			uri: `http://127.0.0.1:${server.address().port}/2010-04-01/ping.json`,
		});
		assert.equal(response.statusCode, 200);
		// twilio hands back axios's parsed JSON body.
		assert.deepEqual(response.body, { ok: true });
		assert.equal(seen.method, "GET");
		const expected = "Basic " + Buffer.from("AC00000000000000000000000000000000:dummy-token").toString("base64");
		assert.equal(seen.auth, expected);
	} finally {
		server.close();
	}
	// The copy twilio actually resolves must be the patched one.
	const axiosVersion = createRequire(require.resolve("twilio"))("axios/package.json").version;
	const [maj, min] = axiosVersion.split(".").map(Number);
	assert.ok(maj > 1 || (maj === 1 && min >= 20), `twilio resolves axios ${axiosVersion}, expected >= 1.20.0`);
});
