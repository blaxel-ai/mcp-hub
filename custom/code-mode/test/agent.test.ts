import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { createMCPClient } from "@ai-sdk/mcp";
import { generateText, stepCountIs } from "ai";
import { openai } from "@ai-sdk/openai";
import { ChildProcess, spawn } from "node:child_process";

const PORT = 9877;
const MCP_URL = `http://localhost:${PORT}/mcp`;

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
        body: JSON.stringify({ jsonrpc: "2.0", method: "initialize", id: 0, params: { capabilities: {}, protocolVersion: "2025-03-26", clientInfo: { name: "test", version: "1.0.0" } } }),
      });
      if (res.ok) return;
    } catch {}
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`Server at ${url} did not become ready within ${timeoutMs}ms`);
}

beforeAll(async () => {
  serverProcess = spawn("npx", ["tsx", "src/server.ts"], {
    cwd: new URL("..", import.meta.url).pathname,
    env: { ...process.env, PORT: String(PORT) },
    stdio: "pipe",
  });

  serverProcess.stderr?.on("data", (data) => {
    process.stderr.write(`[server] ${data}`);
  });
  serverProcess.stdout?.on("data", (data) => {
    process.stdout.write(`[server] ${data}`);
  });

  await waitForServer(MCP_URL);
});

afterAll(async () => {
  serverProcess?.kill("SIGTERM");
  await new Promise((r) => setTimeout(r, 500));
});

describe("code-mode agent", () => {
  it("should use search tool to find endpoints in the API reference", async () => {
    const client = await createMCPClient({
      transport: { type: "http", url: MCP_URL },
    });

    try {
      const tools = await client.tools();
      expect(tools).toHaveProperty("search");
      expect(tools).toHaveProperty("execute");

      const { text, steps } = await generateText({
        model: openai("gpt-5.2"),
        tools,
        stopWhen: stepCountIs(5),
        prompt:
          "List my agents, sandboxes and functions and jobs",
      });

      console.log(text)
      const toolCalls = steps.flatMap((s) => s.toolCalls);
      expect(toolCalls.length).toBeGreaterThan(0);
      expect(toolCalls.some((tc) => tc.toolName === "search")).toBe(true);
      expect(text.length).toBeGreaterThan(0);

      console.log(`Agent response (${toolCalls.length} tool calls):\n${text}`);
    } finally {
      await client.close();
    }
  });
});
