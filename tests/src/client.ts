import { exit } from "process";
import { name, payload, url } from "./config";

import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";

async function connectClient(baseUrl: string): Promise<Client> {
	const client = new Client({ name: `tests-${name}`, version: "1.0.0" });
	const endpoint = new URL("/mcp", baseUrl);
	const transport = new StreamableHTTPClientTransport(endpoint);
	const result = await client.connect(transport);
	console.log(result);
	return client;
}

const main = async () => {
		const client = await connectClient(url);
		console.log("Client initialized");
		// List tools
		const toolsResponse = await client.listTools();
		console.log("Tools listed");
		const tools = toolsResponse?.tools ?? [];
		for (const tool of tools) {
			let params: string[] = [];
			const props = (tool.inputSchema as any)?.properties;
			if (props && typeof props === "object") {
				params = Object.keys(props);
			}
			console.log(`${tool.name} (${params.join(", ")})`);
		}

		let previousResult: Record<string, any> = {};
		for (const fn of payload) {
			const params = fn(previousResult);
			const exists = tools.find((t) => t.name === params.name);
			if (!exists) {
				console.error(`Tool with name ${params.name} not found`);
				exit(1);
			}
			try {
				const result = await client.callTool({ name: params.name, arguments: params.arguments });

				// Result content follows MCP content schema
				const content = result?.content ?? [];
				if (Array.isArray(content) && content.length > 0) {
					const first = content[0] as any;
					if (first.text !== undefined) {
						try {
							previousResult[params.name] = JSON.parse(first.text);
						} catch {
							previousResult[params.name] = first.text;
						}
						console.log(`Result: ${params.name}`, previousResult[params.name]);
					} else {
						console.log(`Result: ${params.name}`, content);
					}
				} else {
					console.log(`Result: ${params.name}`, result);
				}
			} catch (error) {
				console.log(`Error: ${params.name}`, error);
			}
		}
		exit(0);
};

main();
