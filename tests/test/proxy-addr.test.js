"use strict";

// Smoke test for the `proxy-addr` bump to 2.0.8 (GHSA-jqcg-44mw-7w3h),
// pulled in transitively via @modelcontextprotocol/sdk -> express -> proxy-addr.
//
// express is not a direct dependency of this project, so it is loaded the way
// the MCP SDK resolves it (createRequire from the SDK's own location), not as
// a hoisted phantom. Express computes `req.ip` through proxy-addr.
//
// The advisory: a trust subnet written as an IPv4-mapped IPv6 block with a
// short prefix (`::ffff:10.0.0.0/8`) compiled with all-zero leading bits and
// trusted every IPv4 peer, so `req.ip` returned whatever the client put in
// X-Forwarded-For. On 2.0.8 a 127.0.0.1 peer is not in that subnet, so the
// header is ignored.
const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const { createRequire } = require("node:module");

const sdkRequire = createRequire(require.resolve("@modelcontextprotocol/sdk/server/index.js"));
const express = sdkRequire("express");

async function ipSeenWith(trust) {
	const app = express();
	app.set("trust proxy", trust);
	app.get("/ip", (req, res) => res.send(req.ip));
	const server = app.listen(0, "127.0.0.1");
	await new Promise((resolve) => server.once("listening", resolve));
	try {
		return await new Promise((resolve, reject) => {
			http.get(
				{ host: "127.0.0.1", port: server.address().port, path: "/ip", headers: { "X-Forwarded-For": "203.0.113.7" } },
				(res) => {
					let body = "";
					res.on("data", (c) => (body += c));
					res.on("end", () => resolve(body));
				},
			).on("error", reject);
		});
	} finally {
		server.close();
	}
}

test("a trusted loopback proxy still forwards the client IP", async () => {
	assert.equal(await ipSeenWith("loopback"), "203.0.113.7");
});

test("a short-prefix IPv4-mapped trust subnet does not trust every IPv4 peer (GHSA-jqcg-44mw-7w3h)", async () => {
	assert.equal(await ipSeenWith("::ffff:10.0.0.0/8"), "127.0.0.1");
});
