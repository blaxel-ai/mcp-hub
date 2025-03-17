#!/usr/bin/env node
import { Tool } from '@modelcontextprotocol/sdk/types.js';
import OpenAI from 'openai';

interface GenerateImageArgs {
	prompt: string;
	model: string;
	n: number;
	size: string;
}

// Tool definitions
const generateImageTool: Tool = {
	name: 'generate_image',
	description: `Generate high-quality images from text descriptions using DALL-E.
Ideal for creating custom illustrations, artwork, or visualizations.
Provide detailed prompts including style (e.g., 'photorealistic', 'cartoon', 'oil painting'), subject matter, composition, lighting, and color preferences for best results.

The tool returns an image URL
You should format appropriately:
  - use markdown syntax ![description](url) for markdown contexts
  - use HTML <img> tags for emails and html contexts <img src=\"url\" alt=\"description\" style=\"max-width: 100%;\">.
`,
	inputSchema: {
		type: 'object',
		properties: {
			prompt: {
				type: 'string',
				description: 'A detailed description of the image you want to generate. Be specific about style, content, and composition.',
				minLength: 1,
				maxLength: 4000,
			},
			imageName: {
				type: 'string',
				description: 'The filename for the image excluding any extensions.',
				default: 'image',
				pattern: '^[a-zA-Z0-9-_]+$',
				minLength: 1,
				maxLength: 100,
			},
			size: {
				type: 'string',
				description: 'The size of the generated image. Larger sizes provide more detail but may take longer to process.',
				enum: ['1024x1024'],
				default: '1024x1024',
			},
		},
		required: ['prompt'],
		additionalProperties: false,
	},
};

class DallEClient {
	private client: OpenAI;
	IMAGE_MODEL = 'dall-e-3';

	constructor(config: Record<string, string>, secrets: Record<string, string>) {
		this.client = new OpenAI({
			apiKey: secrets.apiKey,
		});
	}

	async generateImage(request: GenerateImageArgs): Promise<any> {
		try {
			const result = await this.client.images.generate({
				model: this.IMAGE_MODEL,
				prompt: request.prompt,
				response_format: 'url',
				size: (request.size as any) || '1024x1024',
			});
			return {
				content: [{ type: 'text', text: result.data[0].url }],
				isError: false,
			};
		} catch (error) {
			return {
				content: [{ type: 'text', text: JSON.stringify(error, null, 2) }],
				isError: true,
			};
		}
	}
}

export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	try {
		const dallEClient = new DallEClient(config, secrets);
		const requestBody: { name: string; arguments: any } = (await request.json()) as { name: string; arguments: any };
		if (!requestBody.arguments) {
			throw new Error('No arguments provided');
		}

		switch (requestBody.name) {
			case 'generate_image': {
				const args = requestBody.arguments as GenerateImageArgs;
				return await dallEClient.generateImage(args);
			}

			default:
				throw new Error(`Unknown tool: ${requestBody.name}`);
		}
	} catch (error) {
		return {
			content: [
				{
					type: 'text',
					text: JSON.stringify({
						error: error instanceof Error ? error.message : String(error),
					}),
				},
			],
		};
	}
}

export async function list() {
	return {
		tools: [generateImageTool],
	};
}

export async function infos() {
	return {
		name: 'dall-e',
		displayName: 'Dall-E',
		integration: 'dall-e',
		categories: ['image'],
		description: 'Generate images with the OpenAI Dall-E model',
		icon: 'data:image/svg+xml;base64,PHN2ZyB2aWV3Qm94PSIwIDAgMzIwIDMyMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cGF0aCBkPSJtMjk3LjA2IDEzMC45N2M3LjI2LTIxLjc5IDQuNzYtNDUuNjYtNi44NS02NS40OC0xNy40Ni0zMC40LTUyLjU2LTQ2LjA0LTg2Ljg0LTM4LjY4LTE1LjI1LTE3LjE4LTM3LjE2LTI2Ljk1LTYwLjEzLTI2LjgxLTM1LjA0LS4wOC02Ni4xMyAyMi40OC03Ni45MSA1NS44Mi0yMi41MSA0LjYxLTQxLjk0IDE4LjctNTMuMzEgMzguNjctMTcuNTkgMzAuMzItMTMuNTggNjguNTQgOS45MiA5NC41NC03LjI2IDIxLjc5LTQuNzYgNDUuNjYgNi44NSA2NS40OCAxNy40NiAzMC40IDUyLjU2IDQ2LjA0IDg2Ljg0IDM4LjY4IDE1LjI0IDE3LjE4IDM3LjE2IDI2Ljk1IDYwLjEzIDI2LjggMzUuMDYuMDkgNjYuMTYtMjIuNDkgNzYuOTQtNTUuODYgMjIuNTEtNC42MSA0MS45NC0xOC43IDUzLjMxLTM4LjY3IDE3LjU3LTMwLjMyIDEzLjU1LTY4LjUxLTkuOTQtOTQuNTF6bS0xMjAuMjggMTY4LjExYy0xNC4wMy4wMi0yNy42Mi00Ljg5LTM4LjM5LTEzLjg4LjQ5LS4yNiAxLjM0LS43MyAxLjg5LTEuMDdsNjMuNzItMzYuOGMzLjI2LTEuODUgNS4yNi01LjMyIDUuMjQtOS4wN3YtODkuODNsMjYuOTMgMTUuNTVjLjI5LjE0LjQ4LjQyLjUyLjc0djc0LjM5Yy0uMDQgMzMuMDgtMjYuODMgNTkuOS01OS45MSA1OS45N3ptLTEyOC44NC01NS4wM2MtNy4wMy0xMi4xNC05LjU2LTI2LjM3LTcuMTUtNDAuMTguNDcuMjggMS4zLjc5IDEuODkgMS4xM2w2My43MiAzNi44YzMuMjMgMS44OSA3LjIzIDEuODkgMTAuNDcgMGw3Ny43OS00NC45MnYzMS4xYy4wMi4zMi0uMTMuNjMtLjM4LjgzbC02NC40MSAzNy4xOWMtMjguNjkgMTYuNTItNjUuMzMgNi43LTgxLjkyLTIxLjk1em0tMTYuNzctMTM5LjA5YzctMTIuMTYgMTguMDUtMjEuNDYgMzEuMjEtMjYuMjkgMCAuNTUtLjAzIDEuNTItLjAzIDIuMnY3My42MWMtLjAyIDMuNzQgMS45OCA3LjIxIDUuMjMgOS4wNmw3Ny43OSA0NC45MS0yNi45MyAxNS41NWMtLjI3LjE4LS42MS4yMS0uOTEuMDhsLTY0LjQyLTM3LjIyYy0yOC42My0xNi41OC0zOC40NS01My4yMS0yMS45NS04MS44OXptMjIxLjI2IDUxLjQ5LTc3Ljc5LTQ0LjkyIDI2LjkzLTE1LjU0Yy4yNy0uMTguNjEtLjIxLjkxLS4wOGw2NC40MiAzNy4xOWMyOC42OCAxNi41NyAzOC41MSA1My4yNiAyMS45NCA4MS45NC03LjAxIDEyLjE0LTE4LjA1IDIxLjQ0LTMxLjIgMjYuMjh2LTc1LjgxYy4wMy0zLjc0LTEuOTYtNy4yLTUuMi05LjA2em0yNi44LTQwLjM0Yy0uNDctLjI5LTEuMy0uNzktMS44OS0xLjEzbC02My43Mi0zNi44Yy0zLjIzLTEuODktNy4yMy0xLjg5LTEwLjQ3IDBsLTc3Ljc5IDQ0Ljkydi0zMS4xYy0uMDItLjMyLjEzLS42My4zOC0uODNsNjQuNDEtMzcuMTZjMjguNjktMTYuNTUgNjUuMzctNi43IDgxLjkxIDIyIDYuOTkgMTIuMTIgOS41MiAyNi4zMSA3LjE1IDQwLjF6bS0xNjguNTEgNTUuNDMtMjYuOTQtMTUuNTVjLS4yOS0uMTQtLjQ4LS40Mi0uNTItLjc0di03NC4zOWMuMDItMzMuMTIgMjYuODktNTkuOTYgNjAuMDEtNTkuOTQgMTQuMDEgMCAyNy41NyA0LjkyIDM4LjM0IDEzLjg4LS40OS4yNi0xLjMzLjczLTEuODkgMS4wN2wtNjMuNzIgMzYuOGMtMy4yNiAxLjg1LTUuMjYgNS4zMS01LjI0IDkuMDZsLS4wNCA4OS43OXptMTQuNjMtMzEuNTQgMzQuNjUtMjAuMDEgMzQuNjUgMjB2NDAuMDFsLTM0LjY1IDIwLTM0LjY1LTIweiIvPjwvc3ZnPg==',
		url: 'https://platform.openai.com/settings/organization/api-keys',
		form: {
			config: {},
			secrets: {
				apiKey: {
					description: 'OpenAI API key',
					label: 'OpenAI API key',
					required: true,
				},
			},
		},
	};
}
