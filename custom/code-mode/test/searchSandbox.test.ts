import { describe, it, expect } from "vitest";
import { evaluateSearch } from "../src/searchSandbox.js";

const spec = {
  info: { title: "Pet Store", version: "1.0.0" },
  servers: [{ url: "https://api.example.com" }],
  paths: {
    "/pets": {
      get: { summary: "List pets", tags: ["pets"] },
      post: { summary: "Create pet", tags: ["pets"] },
    },
    "/users": {
      get: { summary: "List users", tags: ["users"] },
    },
  },
};

describe("evaluateSearch (spec traversal)", () => {
  it("returns keys of spec.paths", async () => {
    const result = await evaluateSearch(spec, "() => Object.keys(spec.paths)");
    expect(result).toEqual(["/pets", "/users"]);
  });

  it("supports async arrow functions that iterate the spec by tag", async () => {
    const code = `async () => {
      const results = [];
      for (const [path, methods] of Object.entries(spec.paths)) {
        for (const [method, op] of Object.entries(methods)) {
          if (op.tags?.some(t => t.toLowerCase() === 'pets')) {
            results.push({ method: method.toUpperCase(), path });
          }
        }
      }
      return results;
    }`;
    const result = await evaluateSearch(spec, code);
    expect(result).toEqual([
      { method: "GET", path: "/pets" },
      { method: "POST", path: "/pets" },
    ]);
  });

  it("exposes info and servers on the spec", async () => {
    const result = await evaluateSearch(spec, "() => ({ title: spec.info.title, url: spec.servers[0].url })");
    expect(result).toEqual({ title: "Pet Store", url: "https://api.example.com" });
  });

  it("normalises undefined returns to null", async () => {
    const result = await evaluateSearch(spec, "() => undefined");
    expect(result).toBeNull();
  });

  it("normalises unserializable returns (function) to null without an opaque error", async () => {
    expect(await evaluateSearch(spec, "() => () => 1")).toBeNull();
  });

  it("surfaces a clear error (not an opaque internal one) for BigInt returns", async () => {
    await expect(evaluateSearch(spec, "() => 10n")).rejects.toThrow(/BigInt/i);
  });
});

describe("evaluateSearch (isolation / credential theft prevention)", () => {
  it("does not expose process (BL_* env vars unreachable)", async () => {
    const result = await evaluateSearch(spec, "() => typeof process");
    expect(result).toBe("undefined");
  });

  it("does not expose require", async () => {
    const result = await evaluateSearch(spec, "() => typeof require");
    expect(result).toBe("undefined");
  });

  it("does not expose fetch or host globals", async () => {
    const result = await evaluateSearch(
      spec,
      "() => [typeof fetch, typeof globalThis.fetch, typeof Buffer]",
    );
    expect(result).toEqual(["undefined", "undefined", "undefined"]);
  });

  it("rejects dynamic import() attempts", async () => {
    await expect(
      evaluateSearch(
        spec,
        `async () => { const cp = await import("node:child_process"); return cp.execSync("id").toString(); }`,
      ),
    ).rejects.toThrow();
  });

  it("cannot read host env via a constructor escape", async () => {
    const result = await evaluateSearch(
      spec,
      `() => { try { return this.constructor.constructor("return typeof process")(); } catch { return "blocked"; } }`,
    );
    expect(["undefined", "blocked"]).toContain(result);
  });

  it("terminates runaway loops via the timeout", async () => {
    await expect(
      evaluateSearch(spec, "() => { while (true) {} }", 200),
    ).rejects.toThrow();
  });

  it("rejects a never-settling promise within the timeout (no unbounded await)", async () => {
    const start = Date.now();
    await expect(
      evaluateSearch(spec, "() => new Promise(() => {})", 200),
    ).rejects.toThrow(/timed out/i);
    expect(Date.now() - start).toBeLessThan(2_000);
  });
});
