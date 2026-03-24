import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";

const BASE_URL = process.env.BL_SEARCH_URL || "http://localhost:1400";
const CONCURRENCY = 20;
// If requests are sequential, total time >= CONCURRENCY * single request time.
// We allow total time up to 3x a single request time to confirm parallelism.
const PARALLELISM_FACTOR = 3;

async function createClient(): Promise<Client> {
  const client = new Client({ name: "test-blaxel-search-concurrent", version: "1.0.0" });
  const endpoint = new URL("/mcp", BASE_URL);
  const transport = new StreamableHTTPClientTransport(endpoint, {
    requestInit: {
      headers: {
        "Authorization": `Bearer ${process.env.BL_API_KEY}`,
        "X-Blaxel-Workspace": process.env.BL_WORKSPACE,
      }
    }
  });
  await client.connect(transport);
  return client;
}

async function main() {
  console.log(`Connecting to blaxel-search at ${BASE_URL}...`);
  const client = await createClient();
  console.log("Client connected");

  // Verify the tool exists
  const { tools } = await client.listTools();
  const searchTool = tools.find((t) => t.name === "web_search_exa");
  if (!searchTool) {
    console.error("Tool web_search_exa not found. Available tools:", tools.map((t) => t.name).join(", "));
    process.exit(1);
  }
  console.log("Tool web_search_exa found");

  // Warm up with a single request to get baseline timing
  console.log("Warming up with a single request...");
  const warmupStart = Date.now();
  await client.callTool({
    name: "web_search_exa",
    arguments: { query: "What is the capital of France?" },
  });
  const singleRequestTime = Date.now() - warmupStart;
  console.log(`Single request took ${singleRequestTime}ms`);

  // Fire 20 concurrent requests
  const queries = [
    "What is the capital of Germany?",
    "What is the capital of Spain?",
    "What is the capital of Italy?",
    "What is the capital of Portugal?",
    "What is the capital of Netherlands?",
    "What is the capital of Belgium?",
    "What is the capital of Austria?",
    "What is the capital of Sweden?",
    "What is the capital of Norway?",
    "What is the capital of Denmark?",
    "What is the capital of Finland?",
    "What is the capital of Poland?",
    "What is the capital of Czech Republic?",
    "What is the capital of Hungary?",
    "What is the capital of Romania?",
    "What is the capital of Greece?",
    "What is the capital of Ireland?",
    "What is the capital of Switzerland?",
    "What is the capital of Croatia?",
    "What is the capital of Bulgaria?",
  ];

  console.log(`\nSending ${CONCURRENCY} concurrent requests...`);
  const concurrentStart = Date.now();

  const results = await Promise.all(
    queries.map((query) =>
      client.callTool({
        name: "web_search_exa",
        arguments: { query },
      })
    )
  );
  const concurrentTime = Date.now() - concurrentStart;
  console.log(`${CONCURRENCY} concurrent requests took ${concurrentTime}ms`);

  // Validate all requests succeeded
  let successCount = 0;
  for (let i = 0; i < results.length; i++) {
    const content = results[i]?.content;
    if (Array.isArray(content) && content.length > 0) {
      successCount++;
    } else {
      console.error(`Request ${i} (${queries[i]}) returned empty content`);
    }
  }
  console.log(`${successCount}/${CONCURRENCY} requests succeeded`);

  if (successCount !== CONCURRENCY) {
    console.error("FAIL: Not all requests succeeded");
    process.exit(1);
  }

  // Check parallelism: concurrent time should be much less than CONCURRENCY * single request time
  const sequentialEstimate = singleRequestTime * CONCURRENCY;
  const maxAllowedTime = singleRequestTime * PARALLELISM_FACTOR;

  console.log(`\n--- Parallelism check ---`);
  console.log(`Single request:      ${singleRequestTime}ms`);
  console.log(`Sequential estimate: ${sequentialEstimate}ms (${CONCURRENCY} x ${singleRequestTime}ms)`);
  console.log(`Concurrent actual:   ${concurrentTime}ms`);
  console.log(`Max allowed:         ${maxAllowedTime}ms (${PARALLELISM_FACTOR}x single request)`);

  if (concurrentTime > maxAllowedTime) {
    console.error(
      `FAIL: Requests appear to be sequential. Concurrent time (${concurrentTime}ms) > ${PARALLELISM_FACTOR}x single request time (${maxAllowedTime}ms)`
    );
    process.exit(1);
  }

  console.log("PASS: Requests executed in parallel");
  process.exit(0);
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
