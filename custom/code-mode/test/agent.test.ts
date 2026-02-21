import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { query } from "@anthropic-ai/claude-agent-sdk";
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
    let toolCallCount = 0;
    let resultText = "";

    for await (const message of query({
      prompt: "List my agents, sandboxes and functions and jobs",
      options: {
        model: 'claude-sonnet-4-6',
        maxTurns: 5,
        mcpServers: {
          "code-mode": {
            type: "http",
            url: MCP_URL,
          },
        },
        allowedTools: ["mcp__code-mode__*"],
      },
    })) {
      if (message.type === "assistant") {
        for (const block of message.message.content) {
          if (block.type === "tool_use") {
            toolCallCount++;
          }
        }
      }

      if (message.type === "result" && message.subtype === "success") {
        resultText = message.result;
      }
    }

    console.log(`Agent response (${toolCallCount} tool calls):\n${resultText}`);

    expect(toolCallCount).toBeGreaterThan(0);
    expect(resultText.length).toBeGreaterThan(0);
  });
});
