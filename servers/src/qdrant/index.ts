import { Embeddings } from './embeddings.js';
import { QdrantConnector } from './qdrant.js';

export const infos = async () => {
	return {
		name: 'qdrant',
		displayName: 'Qdrant',
		categories: ['search'],
		integration: 'qdrant',
		description: 'Store and retrieve memories using Qdrant',
		icon: 'https://avatars.githubusercontent.com/u/73504361?s=200&v=4',
		url: 'https://qdrant.tech/documentation/cloud/authentication/',
		form: {
			config: {
				url: {
					description: 'Qdrant URL',
					label: 'Qdrant URL',
					required: true,
				},
				collectionName: {
					description: 'Collection Name',
					label: 'Collection Name',
					required: true,
				},
				embeddingModel: {
					description: 'Embedding Model',
					label: 'Embedding Model',
					type: 'model-embedding',
					required: true,
				},
				embeddingModelType: {
					description: 'Embedding Model Type',
					label: 'Embedding Model Type',
					type: 'string',
					required: true,
				},
				blWorkspace: {
					description: 'Blaxel Workspace',
					label: 'Blaxel Workspace',
					required: true,
				},
			},
			secrets: {
				apiKey: {
					description: 'API Key',
					label: 'API Key',
					required: true,
				},
				blClientCredentials: {
					description: 'Client Credentials',
					label: 'Client Credentials',
					required: true,
					hidden: true,
				},
			},
		},
	};
};

export const list = async () => {
	return {
		tools: [
			{
				name: 'qdrant_store_memory',
				description: 'Keep the memory for later use, when you are asked to remember something.',
				inputSchema: {
					type: 'object',
					properties: {
						query: {
							type: 'string',
						},
					},
					required: ['query'],
				},
			},
			{
				name: 'qdrant_find_memories',
				description:
					'Look up memories in Qdrant. Use this tool when you need to:\n' +
					' - Find memories by their content\n' +
					' - Access memories for further analysis\n' +
					' - Get some personal information about the user',
				inputSchema: {
					type: 'object',
					properties: {
						query: {
							type: 'string',
							description: 'The query to search for in the memories',
						},
						limit: {
							type: 'number',
							description: 'The number of memories to return (default 10)',
							default: 10,
						},
						scoreThreshold: {
							type: 'number',
							description: 'The score threshold for the memories to return (default 0.5)',
							default: 0.5,
						},
					},
					required: ['query'],
				},
			},
		],
	};
};

export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	try {
		const body: { name: string; arguments: Record<string, any> } = await request.json();

		const { name, arguments: args } = body;
		const qdrantConnector = new QdrantConnector({
			url: config.url,
			apiKey: secrets.apiKey,
			collectionName: config.collectionName,
		});

		if (!args) {
			throw new Error('No arguments provided');
		}
		const { query } = args;
		if (!query) {
			throw new Error('Query is required');
		}

		const baseUrl = process.env.BL_ENV == 'dev' ? 'https://api.blaxel.dev/v0' : 'https://api.blaxel.ai/v0';
		const runUrl = process.env.BL_ENV == 'dev' ? 'https://run.blaxel.dev' : 'https://run.blaxel.ai';
		const embeddings = new Embeddings({
			model: config.embeddingModel,
			modelType: config.embeddingModelType || 'openai',
			clientCredentials: secrets.blClientCredentials,
			baseUrl,
			runUrl,
			workspace: config.blWorkspace,
		});
		switch (name) {
			case 'qdrant_store_memory': {
				const embedding = await embeddings.embed(query);
				await qdrantConnector.storeMemory(embedding as number[], query);
				return {
					content: [{ type: 'text', text: `Remembered` }],
					isError: false,
				};
			}
			case 'qdrant_find_memories': {
				const { limit = 10, scoreThreshold = 0.5 } = args;
				const embedding = await embeddings.embed(query);
				const memories = await qdrantConnector.findMemory(embedding as number[], limit, scoreThreshold);
				return {
					content: memories.points.map((point: any) => ({
						type: 'text',
						text: JSON.stringify(point),
					})),
					isError: false,
				};
			}
			default:
				throw new Error(`Unknown tool: ${name}`);
		}
	} catch (error) {
		let content: String;
		try {
			content = JSON.stringify({
				error: error instanceof Error ? error.message : String(error),
			});
		} catch (error) {
			content = String(error);
		}
		return {
			isError: true,
			content: [
				{
					type: 'text',
					text: content,
				},
			],
		};
	}
}
