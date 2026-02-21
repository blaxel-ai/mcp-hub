import { createFunction, getFunction, deleteFunction, settings } from "@blaxel/core";
import { query } from "@anthropic-ai/claude-agent-sdk";
import dotenv from "dotenv";

dotenv.config();

const POLL_INTERVAL_MS = 5_000;
const DEPLOY_TIMEOUT_MS = 300_000;

function randomName(): string {
  const adjectives = ["swift", "bright", "calm", "bold", "keen", "warm"];
  const nouns = ["falcon", "panda", "otter", "lynx", "raven", "reef"];
  const adj = adjectives[Math.floor(Math.random() * adjectives.length)];
  const noun = nouns[Math.floor(Math.random() * nouns.length)];
  const suffix = Math.random().toString(36).slice(2, 6);
  const name = `cm-test-${adj}-${noun}-${suffix}`;
  if (name.length > 48) throw new Error(`Generated name exceeds 48 chars: ${name}`);
  return name;
}

async function waitForDeployed(functionName: string): Promise<any> {
  const start = Date.now();
  while (Date.now() - start < DEPLOY_TIMEOUT_MS) {
    const { data: fn } = await getFunction({ path: { functionName } });
    console.log(`  status: ${fn?.status} (${Math.round((Date.now() - start) / 1000)}s)`);
    if (fn?.status === "DEPLOYED") return fn;
    if (fn?.status === "FAILED") {
      throw new Error(`Function "${functionName}" deployment failed`);
    }
    await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
  }
  throw new Error(`Function "${functionName}" did not deploy within ${DEPLOY_TIMEOUT_MS / 1000}s`);
}

async function main() {
  const functionName = randomName();
  console.log(`Creating function: ${functionName}`);

  await createFunction({
    body: {
      metadata: {
        name: functionName,
        displayName: `Integration Test – ${functionName}`,
      },
      spec: {
        runtime: {
          image: "blaxel/code-mode:latest",
          generation: "mk3",
          memory: 2048,
          envs: [
            { name: "OPENAPI_REFERENCE", value: process.env.OPENAPI_REFERENCE! },
            { name: "AUTH_APIKEYAUTH", value: process.env.AUTH_APIKEYAUTH! },
          ],
        },
      },
    },
  });

  console.log("Waiting for deployment...");
  const deployed = await waitForDeployed(functionName);
  console.log(`Deployed! URL: ${deployed.metadata?.url}`);

  const mcpUrl = `${deployed.metadata.url}/mcp`;
  console.log(`Connecting MCP to ${mcpUrl}`);

  let toolCallCount = 0;
  let resultText = "";

  try {
    for await (const message of query({
      prompt: "List my agents, sandboxes and functions",
      options: {
        model: 'claude-sonnet-4-6',
        maxTurns: 5,
        mcpServers: {
          "code-mode": {
            type: "http",
            url: mcpUrl,
            headers: settings.headers as Record<string, string>,
          },
        },
        allowedTools: ["mcp__code-mode__*"],
      },
    })) {
      if (message.type === "system" && message.subtype === "init") {
        console.log(`MCP servers: ${JSON.stringify(message.mcp_servers.map(s => ({ name: s.name, status: s.status })))}`);
        const failed = message.mcp_servers.filter(s => s.status !== "connected");
        if (failed.length > 0) {
          throw new Error(`MCP servers failed to connect: ${JSON.stringify(failed)}`);
        }
      }

      if (message.type === "assistant") {
        for (const block of message.message.content) {
          if (block.type === "tool_use") {
            toolCallCount++;
            console.log(`\n--- Tool call #${toolCallCount}: ${block.name} ---`);
            console.log(`  input: ${JSON.stringify(block.input).slice(0, 200)}`);
          }
          if (block.type === "text") {
            console.log(`  text: ${block.text.slice(0, 300)}${block.text.length > 300 ? "…" : ""}`);
          }
        }
      }

      if (message.type === "result" && message.subtype === "success") {
        resultText = message.result;
      }
    }

    console.log(`\nAgent made ${toolCallCount} tool calls`);
    console.log(`Final response:\n${resultText}`);

    if (toolCallCount === 0) {
      throw new Error("Agent made no tool calls");
    }
  } finally {
    console.log(`\nCleaning up: deleting function ${functionName}`);
    await deleteFunction({ path: { functionName } });
    console.log("Done.");
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
