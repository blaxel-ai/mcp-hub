import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { program } from 'commander';
import { mcpServers } from './servers';
import { transformInZodSchema } from './utils';
import { MCPServer } from "./types";

program
  .version('1.0.0')
  .command('start <name>')
  .description('Start the MCP Hub')
  .action(async (name: string) => {
    // Create an MCP server
	const mcpServer = mcpServers[name] as unknown as MCPServer;
	if (!mcpServer) {
		console.error(`Server ${name} not found`);
		return;
	}

	const server = new McpServer({
		name: name,
		version: "1.0.0"
	});

	const tools = await mcpServer.list()
	
	for (const tool of tools.tools) {
		const zodSchema = transformInZodSchema(tool.inputSchema.properties);
		server.tool(tool.name, tool.description, zodSchema, async (argsSchema) => {
			const config: Record<string, string> = {};
			const secrets: Record<string, string> = {};
			if (mcpServer.infos) {
				const infos = await mcpServer.infos()
				if (!infos) {
					console.error(`Server ${name} infos not found`);
					return {
						content: [],
						isError: true
					}
				}

				for (const key in infos.form.config) {
					config[key] = process.env[transformKeyInEnVarName(key)] || '';
				}
				for (const key in infos.form.secrets) {
					secrets[key] = process.env[transformKeyInEnVarName(key)] || '';
				}
			}

			const requestBody = JSON.stringify({
				name: tool.name,
				arguments: argsSchema
			});

			const request = new Request("http://nothing.com", {
				method: "POST",
				body: requestBody
			});

			const response = await mcpServer.call(request, config, secrets);
			return {
				content: response.content,
				isError: response.isError
			};
		});
	}
    // Start receiving messages on stdin and sending messages on stdout
    const transport = new StdioServerTransport();
    await server.connect(transport);
})

function transformKeyInEnVarName(key: string) {
	// apiKey -> API_KEY
	return key.replace(/([A-Z])/g, '_$1').toUpperCase();
}

program.parse(process.argv);
