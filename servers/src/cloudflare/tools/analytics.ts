import { Tool } from '@modelcontextprotocol/sdk/types.js';
import { ToolHandlers } from '../utils/types';

const ANALYTICS_GET_TOOL: Tool = {
	name: 'analytics_get',
	description: 'Get analytics data from Cloudflare',
	inputSchema: {
		type: 'object',
		properties: {
			zoneId: {
				type: 'string',
				description: 'The zone ID to get analytics for',
			},
			since: {
				type: 'string',
				description: 'Start time for analytics (ISO string)',
			},
			until: {
				type: 'string',
				description: 'End time for analytics (ISO string)',
			},
		},
		required: ['zoneId'],
	},
};

export const ANALYTICS_TOOLS = [ANALYTICS_GET_TOOL];

export const ANALYTICS_HANDLERS: ToolHandlers = {
	analytics_get: async (request, accountId, apiToken) => {
		const { zoneId, since, until } = request.arguments as {
			zoneId: string;
			since?: string;
			until?: string;
		};
		const date = since ? new Date(since).toISOString().split('T')[0] : new Date().toISOString().split('T')[0];

		const graphqlQuery = {
			query: `query {
              viewer {
                  zones(filter: {zoneTag: "${zoneId}"}) {
                      httpRequests1dGroups(
                          limit: 1,
                          filter: {date: "${date}"},
                          orderBy: [date_DESC]
                      ) {
                          dimensions {
                              date
                          }
                          sum {
                              requests
                              bytes
                              threats
                              pageViews
                          }
                          uniq {
                              uniques
                          }
                      }
                  }
              }
          }`,
		};

		const analyticsResponse = await fetch('https://api.cloudflare.com/client/v4/graphql', {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${apiToken}`,
				'Content-Type': 'application/json',
			},
			body: JSON.stringify(graphqlQuery),
		});

		if (!analyticsResponse.ok) {
			throw new Error(`Analytics API error: ${await analyticsResponse.text()}`);
		}

		const analyticsData = await analyticsResponse.json();
		return {
			content: [
				{
					type: 'text',
					text: JSON.stringify(analyticsData, null, 2),
				},
			],
		};
	},
};
