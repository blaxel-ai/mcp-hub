import "dotenv/config";
import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { query } from "@anthropic-ai/claude-agent-sdk";
import { ChildProcess, spawn } from "node:child_process";

const PORT = 9877;
const CODE_MODE_URL = `http://localhost:${PORT}/mcp`;
const PROMPT = "List my agents, sandboxes, functions and jobs";
const MODEL = "claude-sonnet-4-6";
const MAX_TURNS = 5;

// Remove CLAUDECODE to avoid "nested session" error from Claude Code CLI
const cleanEnv = Object.fromEntries(
  Object.entries(process.env).filter(([k]) => k !== "CLAUDECODE"),
) as Record<string, string>;

let serverProcess: ChildProcess;

async function waitForServer(url: string, timeoutMs = 15_000): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json, text/event-stream",
        },
        body: JSON.stringify({
          jsonrpc: "2.0",
          method: "initialize",
          id: 0,
          params: {
            capabilities: {},
            protocolVersion: "2025-03-26",
            clientInfo: { name: "test", version: "1.0.0" },
          },
        }),
      });
      if (res.ok) return;
    } catch {}
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`Server at ${url} did not become ready within ${timeoutMs}ms`);
}

type RunResult = {
  toolCalls: { name: string; input: string }[];
  resultText: string;
  totalCostUsd: number;
  inputTokens: number;
  outputTokens: number;
  cacheReadInputTokens: number;
  cacheCreationInputTokens: number;
  numTurns: number;
  durationMs: number;
  durationApiMs: number;
  modelUsage: Record<string, { inputTokens: number; outputTokens: number; costUSD: number }>;
};

async function runWithMcp(
  mcpServers: Record<string, any>,
  allowedTools: string[],
): Promise<RunResult> {
  const toolCalls: { name: string; input: string }[] = [];
  let resultText = "";
  let totalCostUsd = 0;
  let inputTokens = 0;
  let outputTokens = 0;
  let cacheReadInputTokens = 0;
  let cacheCreationInputTokens = 0;
  let numTurns = 0;
  let durationMs = 0;
  let durationApiMs = 0;
  let modelUsage: Record<string, any> = {};

  for await (const message of query({
    prompt: PROMPT,
    options: {
      model: MODEL,
      maxTurns: MAX_TURNS,
      mcpServers,
      allowedTools,
      env: cleanEnv,
    },
  })) {
    if (message.type === "assistant") {
      for (const block of message.message.content) {
        if (block.type === "tool_use") {
          toolCalls.push({
            name: block.name,
            input: JSON.stringify(block.input).slice(0, 200),
          });
        }
      }
    }

    if (message.type === "result") {
      totalCostUsd = message.total_cost_usd;
      inputTokens = message.usage.input_tokens;
      outputTokens = message.usage.output_tokens;
      cacheReadInputTokens = (message.usage as any).cache_read_input_tokens ?? 0;
      cacheCreationInputTokens = (message.usage as any).cache_creation_input_tokens ?? 0;
      numTurns = message.num_turns;
      durationMs = message.duration_ms;
      durationApiMs = message.duration_api_ms;
      modelUsage = message.modelUsage;

      if (message.subtype === "success") {
        resultText = message.result;
      }
    }
  }

  return {
    toolCalls,
    resultText,
    totalCostUsd,
    inputTokens,
    outputTokens,
    cacheReadInputTokens,
    cacheCreationInputTokens,
    numTurns,
    durationMs,
    durationApiMs,
    modelUsage,
  };
}

function printResult(label: string, r: RunResult) {
  console.log(`\n${"=".repeat(60)}`);
  console.log(`  ${label}`);
  console.log(`${"=".repeat(60)}`);
  console.log(`  Tool calls:    ${r.toolCalls.length}`);
  r.toolCalls.forEach((tc, i) => console.log(`    #${i + 1} ${tc.name} → ${tc.input}`));
  console.log(`  Turns:         ${r.numTurns}`);
  console.log(`  Input tokens:  ${r.inputTokens.toLocaleString()}`);
  console.log(`  Output tokens: ${r.outputTokens.toLocaleString()}`);
  console.log(`  Cache read:    ${r.cacheReadInputTokens.toLocaleString()}`);
  console.log(`  Cache create:  ${r.cacheCreationInputTokens.toLocaleString()}`);
  console.log(`  Total cost:    $${r.totalCostUsd.toFixed(4)}`);
  console.log(`  Duration:      ${(r.durationMs / 1000).toFixed(1)}s (API: ${(r.durationApiMs / 1000).toFixed(1)}s)`);
  console.log(`  Response:      ${r.resultText.slice(0, 200)}${r.resultText.length > 200 ? "…" : ""}`);
}

beforeAll(async () => {
  serverProcess = spawn("npx", ["tsx", "src/server.ts"], {
    cwd: new URL("..", import.meta.url).pathname,
    env: { ...process.env, PORT: String(PORT) },
    stdio: "pipe",
  });
  serverProcess.stderr?.on("data", (d) => process.stderr.write(`[code-mode] ${d}`));
  serverProcess.stdout?.on("data", (d) => process.stdout.write(`[code-mode] ${d}`));
  await waitForServer(CODE_MODE_URL);
});

afterAll(async () => {
  serverProcess?.kill("SIGTERM");
  await new Promise((r) => setTimeout(r, 500));
});

const codeModeServers = {
  "code-mode": { type: "http" as const, url: CODE_MODE_URL },
};
const blaxelServers = {
  blaxel: { command: "blaxel-mcp-server" as const },
};

describe("cost comparison: code-mode vs blaxel MCP", () => {
  it("should compare token usage for the same prompt", async () => {
    // Warm-up runs: populate Anthropic prompt cache for both approaches
    // so the measured runs reflect steady-state costs (cache reads, not creates).
    console.log("\n>>> Warm-up: code-mode MCP...");
    await runWithMcp(codeModeServers, ["mcp__code-mode__*"]);

    console.log("\n>>> Warm-up: blaxel MCP...");
    await runWithMcp(blaxelServers, ["mcp__blaxel__*"]);

    // Measured runs (both with warm caches)
    console.log("\n>>> Measured: code-mode MCP...");
    const codeMode = await runWithMcp(codeModeServers, ["mcp__code-mode__*"]);

    console.log("\n>>> Measured: blaxel MCP...");
    const blaxelMcp = await runWithMcp(blaxelServers, ["mcp__blaxel__*"]);

    printResult("CODE-MODE MCP (search + execute) — warm", codeMode);
    printResult("BLAXEL MCP (all toolsets) — warm", blaxelMcp);

    const codeModeTotal = codeMode.inputTokens + codeMode.outputTokens;
    const blaxelTotal = blaxelMcp.inputTokens + blaxelMcp.outputTokens;
    const tokenDiff = blaxelTotal - codeModeTotal;
    const costDiff = blaxelMcp.totalCostUsd - codeMode.totalCostUsd;

    console.log(`\n${"=".repeat(60)}`);
    console.log("  COMPARISON (warm cache)");
    console.log(`${"=".repeat(60)}`);
    console.log(`  Total tokens — code-mode: ${codeModeTotal.toLocaleString()}, blaxel: ${blaxelTotal.toLocaleString()}`);
    console.log(`  Cache read   — code-mode: ${codeMode.cacheReadInputTokens.toLocaleString()}, blaxel: ${blaxelMcp.cacheReadInputTokens.toLocaleString()}`);
    console.log(`  Cache create — code-mode: ${codeMode.cacheCreationInputTokens.toLocaleString()}, blaxel: ${blaxelMcp.cacheCreationInputTokens.toLocaleString()}`);
    console.log(`  Token diff:    ${tokenDiff > 0 ? "+" : ""}${tokenDiff.toLocaleString()} (${tokenDiff > 0 ? "blaxel uses more" : "code-mode uses more"})`);
    console.log(`  Cost diff:     ${costDiff > 0 ? "+" : ""}$${costDiff.toFixed(4)} (${costDiff > 0 ? "blaxel costs more" : "code-mode costs more"})`);
    console.log(`  Tool calls:    code-mode=${codeMode.toolCalls.length}, blaxel=${blaxelMcp.toolCalls.length}`);

    expect(codeMode.toolCalls.length).toBeGreaterThan(0);
    expect(blaxelMcp.toolCalls.length).toBeGreaterThan(0);
    expect(codeMode.resultText.length).toBeGreaterThan(0);
    expect(blaxelMcp.resultText.length).toBeGreaterThan(0);
  });
});
