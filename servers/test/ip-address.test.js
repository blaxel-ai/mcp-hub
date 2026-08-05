"use strict";

// Smoke test for the `ip-address` dependency bump (GHSA-mwp4-54f8-5fhr /
// CVE-2026-69192), pulled in transitively via
// @modelcontextprotocol/sdk -> express-rate-limit -> ip-address.
//
// `ip-address` is not imported anywhere in this project's own source: it is
// promoted to an explicit devDependency (pinned to the exact patched version)
// purely so this test can require() it directly and pin the resolved
// version, instead of relying on it staying hoisted as a phantom transitive.
//
// The advisory: Address4 accepted a dotted-decimal octet written with a
// leading zero (e.g. "012") and decoded it as decimal (12), while resolvers
// such as the WHATWG URL host parser, inet_aton and getaddrinfo decode the
// same leading zero as octal (10). An app that uses Address4.isPrivate() /
// isLoopback() etc. as an SSRF trust-boundary check can therefore be tricked
// into classifying an internal target as external.
//
// Fixed in ip-address@10.3.1: Address4 now rejects a leading zero outright.
const test = require("node:test");
const assert = require("node:assert/strict");
const { Address4, AddressError } = require("ip-address");

test("ip-address rejects octal-looking leading-zero octets (GHSA-mwp4-54f8-5fhr)", () => {
	assert.throws(() => new Address4("012.0.0.1"), AddressError);
});

test("ip-address still parses well-formed dotted-decimal addresses", () => {
	const addr = new Address4("12.0.0.1");
	assert.equal(addr.correctForm(), "12.0.0.1");
	assert.equal(addr.isPrivate(), false);
});

test("ip-address still classifies private ranges correctly", () => {
	const addr = new Address4("10.0.0.1");
	assert.equal(addr.isPrivate(), true);
});
