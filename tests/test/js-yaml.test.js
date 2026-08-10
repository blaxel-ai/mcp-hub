"use strict";

// Smoke test for the `js-yaml` dependency bump (GHSA-5p4m-2wfm-xmqj), pulled
// in transitively via @jest/globals -> @jest/expect -> jest-snapshot ->
// @jest/transform -> babel-plugin-istanbul -> @istanbuljs/load-nyc-config ->
// js-yaml. That whole chain is Jest's coverage-instrumentation plugin
// (babel-plugin-istanbul); this project's `test:unit` script runs
// `node --test`, not Jest, and there is no babel/jest/nyc config anywhere in
// this package, so the instrumentation code that would call into js-yaml is
// never actually invoked here. This is dev-only, non-runtime exposure.
//
// `js-yaml` is not imported anywhere in this project's own source: it is
// promoted to an explicit devDependency (pinned to the exact patched
// version) purely so this test can require() it directly and pin the
// resolved version, instead of relying on it staying hoisted as a phantom
// transitive.
//
// The advisory: `resolveYamlOmap()` deduplicates `!!omap` keys with
// `objectKeys.indexOf(pairKey)` inside the per-entry loop, an O(n) scan run
// once per entry, making the whole resolution O(n^2). A modestly sized
// YAML `!!omap` document therefore blocks the event loop for seconds inside
// a plain `yaml.load(untrustedInput)` (default schema, no options needed).
// This is the same weakness as CVE-2026-59870 (GHSA-724g-mxrg-4qvm), which
// was fixed in the 5.x line in 5.2.1 but never backported to 3.x/4.x.
//
// Fixed in js-yaml@3.15.1: `lib/js-yaml/type/omap.js` replaces the
// `indexOf`-based array scan with an object-keyed hash lookup, making
// dedup O(1) per entry (verified via
// `gh api /repos/nodeca/js-yaml/compare/3.15.0...3.15.1` - the only line
// changed is that lookup; duplicate-key rejection semantics are preserved).
//
// The fix does not change *what* gets accepted or rejected, only how fast -
// a duplicate key is rejected on both 3.15.0 and 3.15.1, so a
// correctness-only assertion can't tell them apart. The discriminator is
// therefore timing, sized so the margin on *both* sides of the threshold is
// comfortable, not just the ratio between the two regimes. Measured on this
// machine before writing the assertion (best of 3, ms):
//   n        3.15.1 (fixed)   3.15.0 (vulnerable)
//   70,000   ~90-230          ~2,200-2,234
//   100,000  ~111             ~4,527
//   140,000  ~148             ~9,090
//   180,000  ~193             ~15,343
// which matches the advisory's O(n^2) signature (~4x runtime per doubling of
// n on 3.15.0, ~1.3-1.7x on the O(1)-lookup 3.15.1). n=70,000 with a 1000ms
// threshold gave only an ~11x margin above the fixed runtime and ~2.2x below
// the vulnerable one - too tight for a shared CI runner that can run
// 2-5x slower than this machine plus a GC pause on top, which would eat
// most of the fixed side's margin. n=140,000 with a 2000ms threshold gives
// ~13.5x above the fixed runtime (148ms) and ~4.5x below the vulnerable one
// (9,090ms) - both margins comfortable, and the doc is still only ~2.3MB /
// well under a "few hundred MB" memory concern. The timeout is a hang
// detector, not a performance benchmark: min-of-3 runs guards against a
// single slow tick reading as a false pass.
//
// Considered and rejected: an n-vs-2n scaling-ratio assertion instead of an
// absolute bound (theoretically machine-speed independent). Measured it
// directly (best of 5, ms) before deciding:
//   n=35,000/70,000:   3.15.0 ratio ~3.90x   3.15.1 ratio ~2.06x
//   n=70,000/140,000:  3.15.0 ratio ~4.14x   3.15.1 ratio ~2.13x
// A doubling ratio is capped near the O(n^2)-vs-O(n) exponents themselves
// (~4x vs ~2x), so any threshold between them gets only a ~1.3-1.5x margin
// on each side - tighter than the absolute bound above, not looser. A
// bigger multiplier widens it (10x-input ratio measured at ~83-95x
// vulnerable vs ~4.75-9.25x fixed, still only ~3-4x margins either side of
// a threshold) but the fixed side's baseline is then only 10-30ms, noisy
// enough on a shared runner that the ratio itself swings 2x across repeats,
// and the vulnerable side's 10x-larger pathological input costs 9-18s per
// run instead of ~9s. The absolute bound above already passed on the real
// GitHub Actions runner (PR #125, verify-tests, 2026-08-10) - kept it.
const test = require("node:test");
const assert = require("node:assert/strict");
const yaml = require("js-yaml");

function omapDoc(n) {
	const lines = new Array(n);
	for (let i = 0; i < n; i += 1) lines[i] = `- k${i}: ${i}`;
	return `!!omap\n${lines.join("\n")}\n`;
}

test("js-yaml resolves a large !!omap without quadratic blowup (GHSA-5p4m-2wfm-xmqj)", () => {
	const doc = omapDoc(140000);
	let best = Infinity;
	for (let i = 0; i < 3; i += 1) {
		const start = Date.now();
		yaml.load(doc);
		best = Math.min(best, Date.now() - start);
	}
	// Hang detector: fixed version resolves in ~148ms on this machine,
	// vulnerable 3.15.0 takes ~9.1s for the same input. 2000ms leaves a
	// comfortable margin on both sides (~13.5x above fixed, ~4.5x below
	// vulnerable).
	assert.ok(
		best < 2000,
		`omap resolution took ${best}ms for 140k entries, expected well under 2000ms (js-yaml still vulnerable to GHSA-5p4m-2wfm-xmqj?)`,
	);
});

test("js-yaml still rejects a duplicate key in an !!omap", () => {
	assert.throws(() => yaml.load("!!omap\n- a: 1\n- a: 2\n"));
});

test("js-yaml still parses a well-formed !!omap", () => {
	const parsed = yaml.load("!!omap\n- a: 1\n- b: 2\n");
	assert.deepEqual(parsed, [{ a: 1 }, { b: 2 }]);
});
