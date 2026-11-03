// Test-only server for CI's runtime smoke (.github/workflows/mcp-runtime-smoke.yaml). It has no hub/ entry, so
// `mcp-hub import` never builds, pushes or registers it on Blaxel. No secrets and no network: it proves the image,
// super-gateway and the MCP SDKs serve a real tool call after a dependency bump.
import { Tool } from '@modelcontextprotocol/sdk/types.js';
import { ListToolsResponse, ToolResponse } from '../types';

const ECHO_TOOL: Tool = {
	name: 'echo',
	description: 'Returns the text it was given.',
	inputSchema: {
		type: 'object',
		properties: { text: { type: 'string', description: 'Text to return' } },
		required: ['text'],
	},
};

export async function list(): Promise<ListToolsResponse> {
	return { tools: [ECHO_TOOL] };
}

export async function call(request: Request): Promise<ToolResponse> {
	const { name, arguments: args } = (await request.json()) as { name: string; arguments: { text?: string } };
	if (name !== 'echo') {
		return { content: [{ type: 'text', text: `unknown tool ${name}` }], isError: true };
	}
	return { content: [{ type: 'text', text: String(args?.text ?? '') }] };
}

export const infos = async () => ({
	name: 'test-echo',
	displayName: 'Test echo',
	categories: [],
	integration: 'test-echo',
	description: 'CI-only server, never published.',
	form: { config: {}, secrets: {} },
});
