"use strict";

// Smoke test for the `fast-uri` dependency bump (advisory range
// >=3.0.0 <3.1.5), pulled in transitively via
// @modelcontextprotocol/sdk -> ajv-formats -> ajv -> fast-uri.
//
// `fast-uri` is not imported anywhere in this project's own source: it is
// promoted to an explicit devDependency (pinned to the exact patched version)
// purely so this test can require() it directly and pin the resolved
// version, instead of relying on it staying hoisted as a phantom transitive.
//
// The advisory: a URI authority containing a literal backslash (e.g.
// "http://good.com\\evil.com/path") is flagged by `parse()` via its `error`
// field on every affected version - that part never changed. What changed is
// `resolve()`: on the vulnerable range it silently swallows that error and
// still returns a resolved URI string containing the backslash, instead of
// refusing it. A caller relying on `resolve()` throwing to reject a
// malformed/spoofable authority (the pattern ajv's $ref resolution uses)
// stayed exposed even though `parse()` "looked" like it was flagging the
// problem.
//
// Fixed in fast-uri@3.1.5: `resolve()` now throws instead of returning the
// backslash-bearing URI. Verified against 3.1.4 (the version this bump
// replaces) before writing this assertion: 3.1.4's `resolve()` returns
// 'http://good.com\\evil.com/path' without throwing.
const test = require("node:test");
const assert = require("node:assert/strict");
const fastUri = require("fast-uri");

test("fast-uri resolve() rejects a URI authority containing a literal backslash", () => {
	assert.throws(
		() => fastUri.resolve("http://good.com/", "http://good.com\\evil.com/path"),
		/backslash/i,
	);
});

test("fast-uri still resolves well-formed relative references", () => {
	const resolved = fastUri.resolve("http://good.com/a/b", "c/d");
	assert.equal(resolved, "http://good.com/a/c/d");
});
