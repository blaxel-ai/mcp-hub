"use strict";

// Regression test for follow-redirects 1.16.1 (GHSA-8r9p-f939-6h3c, GHSA-q3jh-wrc6-jgq8):
// credential headers must be stripped when a redirect leaves the origin, also for custom
// `sensitiveHeaders` names containing `^` and for array-form headers. follow-redirects reaches
// this project through twilio -> axios, so it is loaded through that chain.
// Both servers listen on loopback; a different port is a different origin for follow-redirects.
const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const { createRequire } = require("node:module");

// Load follow-redirects through the axios twilio resolves (servers/src/twilio's default
// RequestClient), the same copy axios.test.js checks.
function loadFollowRedirects() {
	const fromTwilio = createRequire(require.resolve("twilio"));
	return createRequire(fromTwilio.resolve("axios/package.json"))("follow-redirects");
}

async function listen(handler) {
	const server = http.createServer(handler);
	await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
	return server;
}

async function redirectOnce(options) {
	let seen;
	const target = await listen((req, res) => {
		seen = req.headers;
		res.end("ok");
	});
	const origin = await listen((req, res) => {
		res.writeHead(302, { location: `http://127.0.0.1:${target.address().port}/` });
		res.end();
	});
	try {
		const { http: client } = loadFollowRedirects();
		await new Promise((resolve, reject) => {
			const req = client.request({ host: "127.0.0.1", port: origin.address().port, path: "/", ...options }, (res) => {
				res.resume();
				res.on("end", resolve);
			});
			req.on("error", reject);
			req.end();
		});
	} finally {
		target.close();
		origin.close();
	}
	assert.ok(seen, "the redirect target was reached");
	return seen;
}

test("custom sensitive header containing ^ is stripped on a cross-origin redirect", async () => {
	const seen = await redirectOnce({ headers: { "X-Api^Key": "secret" }, sensitiveHeaders: ["X-Api^Key"] });
	assert.equal(seen["x-api^key"], undefined);
});

test("array-form Authorization header is stripped on a cross-origin redirect", async () => {
	const seen = await redirectOnce({ headers: ["Authorization", "Bearer secret", "X-Trace", "1"] });
	assert.equal(seen.authorization, undefined);
	assert.equal(seen["x-trace"], "1");
});
