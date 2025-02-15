import { ANALYTICS_HANDLERS, ANALYTICS_TOOLS } from './tools/analytics';
import { D1_HANDLERS, D1_TOOLS } from './tools/d1';
import { KV_HANDLERS, KV_TOOLS } from './tools/kv';
import { R2_HANDLERS, R2_TOOLS } from './tools/r2';
import { WORKER_TOOLS, WORKERS_HANDLERS } from './tools/workers';

// Types for Cloudflare responses

// Combine all tools

const ALL_TOOLS = [...KV_TOOLS, ...WORKER_TOOLS, ...ANALYTICS_TOOLS, ...R2_TOOLS, ...D1_TOOLS];

// Handle list tools request
export async function list() {
	return { tools: ALL_TOOLS };
}

// Handle tool calls
export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	const body: { name: string; arguments: Record<string, string> } = (await request.json()) as {
		name: string;
		arguments: Record<string, string>;
	};
	const toolName = body.name;
	const apiToken = secrets.apiToken;
	const accountId = config.accountId;
	try {
		if (toolName in ANALYTICS_HANDLERS) {
			return await ANALYTICS_HANDLERS[toolName](body, accountId, apiToken);
		}
		if (toolName in D1_HANDLERS) {
			return await D1_HANDLERS[toolName](body, accountId, apiToken);
		}
		if (toolName in KV_HANDLERS) {
			return await KV_HANDLERS[toolName](body, accountId, apiToken);
		}
		if (toolName in WORKERS_HANDLERS) {
			return await WORKERS_HANDLERS[toolName](body, accountId, apiToken);
		}
		if (toolName in R2_HANDLERS) {
			return await R2_HANDLERS[toolName](body, accountId, apiToken);
		}

		throw new Error(`Unknown tool: ${toolName}`);
	} catch (error) {
		return {
			content: [
				{
					type: 'text',
					text: `Error: ${error instanceof Error ? error.message : String(error)}`,
				},
			],
			isError: true,
		};
	}
}

export async function infos() {
	return {
		name: 'cloudflare',
		displayName: 'Cloudflare',
		categories: ['analytics', 'storage', 'database'],
		integration: 'cloudflare',
		description: 'Manage your Cloudflare resources (incl. KV, D1, R2, Workers)',
		icon: 'https://qualified-production.s3.us-east-1.amazonaws.com/uploads/4898d5ad5603fcf8e0607d31b7be4a7a7d58c5679929464fa38a3b1562ae7cb0.png',
		url: 'https://dash.cloudflare.com/profile/api-tokens',
		form: {
			config: {
				accountId: {
					description: 'Account ID',
					label: 'Account ID',
					required: true,
				},
			},
			secrets: {
				apiToken: {
					description: 'API Token',
					label: 'API Token',
					required: true,
				},
			},
		},
	};
}
