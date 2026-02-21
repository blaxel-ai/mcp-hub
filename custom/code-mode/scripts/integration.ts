import { createFunction, getFunction, deleteFunction, settings } from "@blaxel/core";
import { createMCPClient } from "@ai-sdk/mcp";
import { generateText, stepCountIs } from "ai";
import { openai } from "@ai-sdk/openai";
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
  console.log(`Connecting MCP client to ${mcpUrl}`);

  const client = await createMCPClient({
    transport: {
      type: "http",
      url: mcpUrl,
      headers: settings.headers,
    },
  });

  try {
    const tools = await client.tools();
    console.log(`Tools: ${Object.keys(tools).join(", ")}`);

    if (!tools.search || !tools.execute) {
      throw new Error(`Expected search and execute tools, got: ${Object.keys(tools).join(", ")}`);
    }

    const { text, steps } = await generateText({
      model: openai("gpt-5.2"),
      tools,
      stopWhen: stepCountIs(5),
      prompt: "List my agents, sandboxes and functions",
      onStepFinish({ stepNumber, finishReason, toolCalls, toolResults, text: stepText }) {
        console.log(`\n--- Step ${stepNumber} (${finishReason}) ---`);
        if (toolCalls.length > 0) {
          for (const tc of toolCalls) {
            console.log(`  tool: ${tc.toolName}(${JSON.stringify(tc.input).slice(0, 200)})`);
          }
        }
        if (toolResults.length > 0) {
          for (const tr of toolResults) {
            const out = typeof tr.output === "string" ? tr.output : JSON.stringify(tr.output);
            console.log(`  result: ${out.slice(0, 300)}${out.length > 300 ? "…" : ""}`);
          }
        }
        if (stepText) {
          console.log(`  text: ${stepText.slice(0, 300)}${stepText.length > 300 ? "…" : ""}`);
        }
      },
    });

    const toolCalls = steps.flatMap((s) => s.toolCalls);
    console.log(`\nAgent made ${toolCalls.length} tool calls across ${steps.length} steps`);
    console.log(`Final response:\n${text}`);

    if (toolCalls.length === 0) {
      throw new Error("Agent made no tool calls");
    }
  } finally {
    await client.close();
    console.log(`\nCleaning up: deleting function ${functionName}`);
    await deleteFunction({ path: { functionName } });
    console.log("Done.");
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
