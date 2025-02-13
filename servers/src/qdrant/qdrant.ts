import { QdrantClient } from '@qdrant/js-client-rest';

export interface QdrantConfig {
	url: string;
	apiKey: string;
	collectionName: string;
}

export class QdrantConnector {
	private client: QdrantClient;

	constructor(private readonly config: QdrantConfig) {
		this.client = new QdrantClient({
			url: config.url,
			apiKey: config.apiKey,
		});
	}

	async createCollection() {
		try {
			const response = await this.client.getCollections();
			if (!response.collections.find((collection: any) => collection.name === this.config.collectionName)) {
				await this.client.createCollection(this.config.collectionName, {
					vectors: {
						default: {
							size: 1536,
							distance: 'Cosine',
						},
					},
				});
			}
		} catch (error) {
			if (error instanceof Error && 'status' in error) {
				throw new Error(`HTTP error creating collection: ${(error as any).status} - ${error.message}`);
			}
			throw new Error(`Error creating collection: ${error}`);
		}
	}

	async storeMemory(embeddings: number[], information: string) {
		try {
			await this.createCollection();
			return await this.client.upsert(this.config.collectionName, {
				wait: true,
				points: [
					{
						id: crypto.randomUUID(),
						vector: {
							default: embeddings,
						},
						payload: { information },
					},
				],
			});
		} catch (error) {
			if (error instanceof Error && 'status' in error) {
				throw new Error(`HTTP error storing memory: ${(error as any).status} - ${error.message}`);
			}
			throw new Error(`Error storing memory: ${error}`);
		}
	}

	async findMemory(query: number[], limit: number = 10) {
		try {
			await this.createCollection();
			return await this.client.query(this.config.collectionName, {
				query,
				using: 'default',
				with_payload: true,
				limit,
			});
		} catch (error) {
			if (error instanceof Error && 'status' in error) {
				throw new Error(`HTTP error finding memory: ${(error as any).status} - ${error.message}`);
			}
			throw new Error(`Error finding memory: ${error}`);
		}
	}
}