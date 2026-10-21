import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";

// A large SigNoz result must fail or succeed on its own; it must never leave
// the server unable to answer the requests that follow it.
const BASE_URL = process.env.SIGNOZ_MCP_URL || "http://localhost:1400";
const TIME_RANGE = process.env.SIGNOZ_LOGS_TIME_RANGE || "24h";
const REQUIRE_OVERSIZED = process.env.REQUIRE_OVERSIZED === "true";
const LARGE_CALL_TIMEOUT_MS = 90_000;
const FOLLOW_UP_TIMEOUT_MS = 15_000;

async function createClient(): Promise<Client> {
  const headers: Record<string, string> = {};
  if (process.env.BL_API_KEY) headers["Authorization"] = `Bearer ${process.env.BL_API_KEY}`;
  if (process.env.BL_WORKSPACE) headers["X-Blaxel-Workspace"] = process.env.BL_WORKSPACE;
  const client = new Client({ name: "test-signoz-large-result", version: "1.0.0" });
  const transport = new StreamableHTTPClientTransport(new URL("/mcp", BASE_URL), { requestInit: { headers } });
  await client.connect(transport);
  return client;
}

function fail(message: string): never {
  console.error(`FAIL: ${message}`);
  process.exit(1);
}

async function main() {
  console.log(`Connecting to SigNoz MCP at ${BASE_URL}...`);
  const client = await createClient();
  const { tools } = await client.listTools();
  if (!tools.some((t) => t.name === "signoz_search_logs")) {
    fail(`signoz_search_logs not found. Available tools: ${tools.map((t) => t.name).join(", ")}`);
  }

  console.log(`Requesting 10000 log rows over ${TIME_RANGE}...`);
  const started = Date.now();
  let oversized = false;
  try {
    const result = await client.callTool(
      { name: "signoz_search_logs", arguments: { timeRange: TIME_RANGE, limit: "10000" } },
      undefined,
      { timeout: LARGE_CALL_TIMEOUT_MS },
    );
    const bytes = Buffer.byteLength(JSON.stringify(result));
    console.log(`Large call returned a result of ${bytes} bytes in ${Date.now() - started}ms`);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    if (!message.includes("exceeded the gateway limit")) {
      fail(`large call did not settle cleanly after ${Date.now() - started}ms: ${message}`);
    }
    oversized = true;
    console.log(`Large call was rejected by the gateway in ${Date.now() - started}ms: ${message}`);
  }

  try {
    await client.ping({ timeout: FOLLOW_UP_TIMEOUT_MS });
    await client.listTools(undefined, { timeout: FOLLOW_UP_TIMEOUT_MS });
  } catch (error) {
    fail(`server stopped answering after the large call: ${error instanceof Error ? error.message : error}`);
  }
  console.log("Server still answers ping and tools/list after the large call");

  if (!oversized) {
    const note = "the result fit under the gateway limit, so the oversized path was not exercised; widen SIGNOZ_LOGS_TIME_RANGE";
    if (REQUIRE_OVERSIZED) fail(note);
    console.warn(`NOTE: ${note}`);
  }
  await client.close();
  console.log("PASS");
}

main().catch((error) => fail(error instanceof Error ? error.message : String(error)));
