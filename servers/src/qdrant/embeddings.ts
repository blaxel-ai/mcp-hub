export interface EmbeddingsConfig {
	model: string;
	modelType: string;
}

export class Embeddings {
	constructor(
		private readonly config: EmbeddingsConfig,
	) {
		this.config = config;
	}

	async embed(query: string): Promise<number[]> {
		switch (this.config.modelType) {
			case 'openai':
				return this.openAIEmbed(query);
			default:
				throw new Error(`Unsupported model type: ${this.config.modelType}`);
		}
	}

	async openAIEmbed(query: string): Promise<number[]> {
		const { model } = this.config;

		const request = new Request(`${process.env.BL_RUN_URL}/models/${model}/v1/embeddings`, {
			method: 'POST',
			body: JSON.stringify({
				input: query,
			}),
			headers: {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${process.env.BL_RUN_API_KEY}`,
			},
		});
		const response = await fetch(request);
		const body = (await response.json()) as { data: [{ embedding: number[] }] };
		return body.data[0].embedding;
	}
}