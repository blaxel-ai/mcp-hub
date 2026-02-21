import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import "@blaxel/telemetry";
import { StreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import express from 'express';
import { z } from 'zod';

// Create an MCP server
const server = new McpServer({
  name: 'GreetingServer',
  version: '1.0.0',
});

server.registerTool(
  'greet',
  {
    description: 'Greet someone by name',
    inputSchema: { name: z.string().optional().default('World')},
    outputSchema: { result: z.string() }
  },
  async ({ name }) => {
    const output = { result: `Hello, ${name}!` };
    return {
      content: [{ type: 'text', text: JSON.stringify(output) }],
      structuredContent: output
    };
  }
);

// Set up Express and HTTP transport
const app = express();
app.use(express.json());

app.post('/mcp', async (req, res) => {
  // Create a new transport for each request to prevent request ID collisions
  const transport = new StreamableHTTPServerTransport({
    sessionIdGenerator: undefined,
    enableJsonResponse: true
  });

  res.on('close', () => {
    transport.close();
  });

  await server.connect(transport);
  await transport.handleRequest(req, res, req.body);
});

const port = parseInt(process.env.PORT || '80');
const host = process.env.HOST || '0.0.0.0';

app.listen(port, () => {
  console.log(`MCP Server running on http://${host}:${port}/mcp`);
}).on('error', error => {
  console.error('Server error:', error);
  process.exit(1);
});