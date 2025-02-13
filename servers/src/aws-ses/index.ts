#!/usr/bin/env node
import { SESClient, SendEmailCommand } from '@aws-sdk/client-ses';
import { Tool } from '@modelcontextprotocol/sdk/types.js';

interface SendEmailArgs {
	from: string;
	to: string;
	subject: string;
	body: string;
}

// Tool definitions
const sendEmailTool: Tool = {
	name: 'send_email',
	description: 'Send an email using AWS SES. The sender email address must be verified in your AWS SES configuration.',
	inputSchema: {
		type: 'object',
		properties: {
			from: {
				type: 'string',
				description: 'The sender email address (must be verified in AWS SES). Example: "no-reply@company.com"',
			},
			to: {
				type: 'string',
				description: 'Recipient email address. Example: "user@example.com"',
			},
			subject: {
				type: 'string',
				description: 'The email subject line',
			},
			body: {
				type: 'string',
				description: 'The plain text content of the email body',
			},
		},
		required: ['from', 'to', 'subject', 'body'],
	},
};

class AwsSesClient {
	private client: SESClient;

	constructor(config: Record<string, string>, secrets: Record<string, string>) {
		this.client = new SESClient({
			region: config.region,
			credentials: {
				accessKeyId: secrets.accessKeyId,
				secretAccessKey: secrets.secretAccessKey,
				sessionToken: secrets.sessionToken,
			},
		});
	}

	async sendEmail(request: SendEmailArgs): Promise<any> {
		try {
			const params = {
				Source: request.from,
				Destination: {
					ToAddresses: [request.to],
				},
				Message: {
					Subject: {
						Data: request.subject,
					},
					Body: {
						Text: {
							Data: request.body,
						},
					},
				},
			};
			const command = new SendEmailCommand(params);
			const result = await this.client.send(command);
			return {
				content: [{ type: 'text', text: JSON.stringify(result, null, 2) }],
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
		const awsSesClient = new AwsSesClient(config, secrets);
		const requestBody: { name: string; arguments: any } = await request.json() as { name: string; arguments: any };
		if (!requestBody.arguments) {
			throw new Error('No arguments provided');
		}

		switch (requestBody.name) {
			case 'send_email': {
				const args = requestBody.arguments as SendEmailArgs;
				await awsSesClient.sendEmail(args);
				return {
					content: [{ type: 'text', text: 'Email sent successfully' }],
					isError: false,
				};
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
		tools: [sendEmailTool],
	};
}

export async function infos() {
	return {
		name: 'aws-ses',
		displayName: 'AWS SES',
		integration: 'aws-ses',
		categories: ['email'],
		description: 'Send emails using AWS SES',
		icon: 'https://a.b.cdn.console.awsstatic.com/a/v1/2QIS3M6GW3A6OS7WHLYZ26DOKTQ3ZGRI22PA57GP4C7Y7ANK5XDQ/icon/f2b32bda85a5a4a613eb47fb01c57ce3-2b4a0b6e3c7d785e7e0d22f5d540dce9.svg',
		url: 'https://aws.amazon.com/ses',
		form: {
			config: {
				region: {
					description: 'AWS region',
					label: 'AWS region',
					required: true,
				},
			},
			secrets: {
				accessKeyId: {
					description: 'AWS access key ID',
					label: 'AWS access key ID',
					required: true,
				},
				secretAccessKey: {
					description: 'AWS secret access key',
					label: 'AWS secret access key',
					required: true,
				},
				sessionToken: {
					description: 'AWS session token',
					label: 'AWS session token',
					required: false,
				},
			},
		},
	};
}
