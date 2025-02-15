#!/usr/bin/env node
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
	description: `
Send an email using Gmail. No sender will be needed, it use a default configuration
You just have to set the recipient as an email string adress inside the parameter "to", a subject and a body in string format.

Example:
{
	"to": "user@example.com",
	"subject": "Hello",
	"body": "This is a test email"
}
`,
	inputSchema: {
		type: 'object',
		properties: {
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
		required: ['to', 'subject', 'body'],
	},
};

class GmailClient {
	private refreshToken: string;
	private accessToken?: string;
	private clientId: string;
	private clientSecret: string;
	private from: string = '';

	constructor(config: Record<string, string>, secrets: Record<string, string>) {
		this.refreshToken = secrets.refreshToken;
		this.clientId = process.env.CLIENT_ID || '';
		this.clientSecret = process.env.CLIENT_SECRET || '';
	}

	private async login() {
		if (this.accessToken) {
			return;
		}

		const response = await fetch('https://oauth2.googleapis.com/token', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
			},
			body: new URLSearchParams({
				client_id: this.clientId,
				client_secret: this.clientSecret,
				refresh_token: this.refreshToken,
				grant_type: 'refresh_token',
			}).toString(),
		});

		if (!response.ok) {
			const error = await response.json();
			throw new Error(`Failed to refresh token: ${JSON.stringify(error)}`);
		}

		const data: { access_token: string; id_token: string } = await response.json() as { access_token: string; id_token: string };
		this.accessToken = data.access_token;

		const decoded = this.decodeJWT(data.id_token);
		this.from = decoded.email;
	}

	private decodeJWT(token: string): any {
		// Get the payload part (second part) of the JWT
		const parts = token.split('.');
		if (parts.length !== 3) {
			throw new Error('Invalid JWT format');
		}

		// Decode the base64 payload
		const payload = parts[1];
		const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
		const jsonPayload = decodeURIComponent(
			atob(base64)
				.split('')
				.map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
				.join(''),
		);

		return JSON.parse(jsonPayload);
	}

	async sendEmail(request: SendEmailArgs): Promise<any> {
		await this.login();
		const message = [`From: "${this.from}" <${this.from}>`, `To: ${request.to}`, `Subject: ${request.subject}`, '', request.body].join(
			'\n',
		);
		console.log(message);

		const encoder = new TextEncoder();
		const bytes = encoder.encode(message);
		const raw = Buffer.from(bytes).toString('base64');

		const response = await fetch('https://www.googleapis.com/gmail/v1/users/me/messages/send', {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${this.accessToken}`,
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({ raw }),
		});
		if (!response.ok) {
			const error = await response.json();
			throw new Error(`Failed to send email: ${JSON.stringify(error)}`);
		}
		return {
			content: [{ type: 'text', text: JSON.stringify(response, null, 2) }],
			isError: false,
		};
	}
}

export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	try {
		const gmailClient = new GmailClient(config, secrets);
		const requestBody: { name: string; arguments: any } = await request.json() as { name: string; arguments: any };
		if (!requestBody.arguments) {
			throw new Error('No arguments provided');
		}

		switch (requestBody.name) {
			case 'send_email': {
				const args = requestBody.arguments as SendEmailArgs;
				return await gmailClient.sendEmail(args);
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
		name: 'gmail',
		displayName: 'Gmail',
		integration: 'gmail',
		categories: ['email'],
		description: 'Send emails using Gmail',
		icon: 'https://upload.wikimedia.org/wikipedia/commons/thumb/7/7e/Gmail_icon_%282020%29.svg/1024px-Gmail_icon_%282020%29.svg.png',
		url: 'https://console.cloud.google.com/apis/credentials',
		form: {
			oauth: {
				type: 'google',
				scope: ['https://www.googleapis.com/auth/gmail.send'],
			},
			config: {},
			secrets: {
				refreshToken: {
					description: 'Google API refresh token',
					label: 'Google API refresh token',
					required: true,
					hidden: true,
				},
			},
		},
	};
}
