import { Tool } from '@modelcontextprotocol/sdk/types.js';
import { ToolResponse } from '../types';
import { isBraveLocalSearchArgs, isBraveWebSearchArgs, performLocalSearch, performWebSearch } from './brave';

export const WEB_SEARCH_TOOL: Tool = {
	name: 'brave_web_search',
	description:
		'Performs a web search using the Brave Search API, ideal for general queries, news, articles, and online content. ' +
		'Use this for broad information gathering, recent events, or when you need diverse web sources. ' +
		'Supports pagination, content filtering, and freshness controls. ' +
		'Maximum 20 results per request, with offset for pagination. ',
	inputSchema: {
		type: 'object',
		properties: {
			query: {
				type: 'string',
				description: 'Search query (max 400 chars, 50 words)',
			},
			count: {
				type: 'number',
				description: 'Number of results (1-20, default 10)',
				default: 10,
			},
			offset: {
				type: 'number',
				description: 'Pagination offset (max 9, default 0)',
				default: 0,
			},
		},
		required: ['query'],
	},
};

export const LOCAL_SEARCH_TOOL: Tool = {
	name: 'brave_local_search',
	description:
		"Searches for local businesses and places using Brave's Local Search API. " +
		'Best for queries related to physical locations, businesses, restaurants, services, etc. ' +
		'Returns detailed information including:\n' +
		'- Business names and addresses\n' +
		'- Ratings and review counts\n' +
		'- Phone numbers and opening hours\n' +
		"Use this when the query implies 'near me' or mentions specific locations. " +
		'Automatically falls back to web search if no local results are found.',
	inputSchema: {
		type: 'object',
		properties: {
			query: {
				type: 'string',
				description: "Local search query (e.g. 'pizza near Central Park')",
			},
			count: {
				type: 'number',
				description: 'Number of results (1-20, default 5)',
				default: 5,
			},
		},
		required: ['query'],
	},
};

export async function call(
	request: Request,
	config: Record<string, string>,
	secrets: Record<string, string>,
	forceBody: { name: string; arguments: Record<string, string> } | null = null,
): Promise<ToolResponse> {
	try {
		const body: { name: string; arguments: Record<string, string> } = forceBody || (await request.json()) as { name: string; arguments: Record<string, string> };
		const { name, arguments: args } = body;

		if (!args) {
			throw new Error('No arguments provided');
		}

		switch (name) {
			case 'brave_web_search': {
				if (!isBraveWebSearchArgs(args)) {
					throw new Error('Invalid arguments for brave_web_search');
				}
				const { query, count = 10 } = args;
				const results = await performWebSearch(query, count, secrets);
				return {
					content: [{ type: 'text', text: results }],
					isError: false,
				};
			}

			case 'brave_local_search': {
				if (!isBraveLocalSearchArgs(args)) {
					throw new Error('Invalid arguments for brave_local_search');
				}
				const { query, count = 5 } = args;
				const results = await performLocalSearch(query, count, secrets);
				return {
					content: [{ type: 'text', text: results }],
					isError: false,
				};
			}

			default:
				return {
					content: [{ type: 'text', text: `Unknown tool: ${name}` }],
					isError: true,
				};
		}
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
