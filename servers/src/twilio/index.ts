#!/usr/bin/env node
import { Tool } from '@modelcontextprotocol/sdk/types.js';
import { Twilio } from 'twilio';

// Type definitions for tool arguments
interface SendSmsArgs {
	to: string;
	message: string;
}

// Tool definition for sending SMS via Twilio
const sendSmsTool: Tool = {
	name: 'send_sms',
	description: 'Send an SMS message using the Twilio API',
	inputSchema: {
		type: 'object',
		properties: {
			to: {
				type: 'string',
				description: 'Recipient phone number in E.164 format',
			},
			message: {
				type: 'string',
				description: 'The SMS message text to send',
			},
		},
		required: ['to', 'message'],
	},
};

/**
 * Main call function.
 * Expects a request body of the form { name: string, arguments: any }.
 * Only the `twilio_send_sms` tool is supported.
 */
export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	try {
		// Parse the incoming request JSON
		const requestBody = (await request.json()) as { name: string; arguments: any };

		if (!requestBody.arguments) {
			throw new Error('No arguments provided');
		}

		switch (requestBody.name) {
			case 'send_sms': {
				const args = requestBody.arguments as SendSmsArgs;
				if (!args.to || !args.message) {
					throw new Error('Missing required parameters: to and message');
				}

				// Retrieve Twilio configuration from secrets
				const accountSid = config.accountSid;
				const fromNumber = config.fromNumber;
				const authToken = secrets.authToken;

				if (!accountSid || !authToken || !fromNumber) {
					throw new Error('Missing Twilio secrets configuration');
				}

				const client = new Twilio(accountSid, authToken);

				// Prepare the payload as URL-encoded parameters
				const params = new URLSearchParams();
				params.append('From', fromNumber);
				params.append('To', args.to);
				params.append('Body', args.message);

				try {
					const fetchResponse = await client.messages.create({
						from: fromNumber,
						to: args.to,
						body: args.message,
					});

					return {
						content: [{ type: 'text', text: `Message sent to ${args.to}` }],
						isError: false,
					};
				} catch (error: any) {
					console.error('Error sending SMS:', error.message);
					return {
						content: [
							{
								type: 'text',
								text: JSON.stringify({ error: error.message }),
							},
						],
						isError: true,
					};
				}
			}
			default:
				throw new Error(`Unknown tool: ${requestBody.name}`);
		}
	} catch (error: any) {
		console.error('Error executing tool:', error);
		return {
			content: [
				{
					type: 'text',
					text: JSON.stringify({
						error: error instanceof Error ? error.message : String(error),
					}),
				},
			],
			isError: true,
		};
	}
}

/**
 * Returns the list of available tools.
 */
export async function list() {
	return {
		tools: [sendSmsTool],
	};
}

/**
 * Returns integration metadata and configuration form.
 */
export async function infos() {
	return {
		name: 'twilio',
		displayName: 'Twilio',
		integration: 'twilio',
		categories: ['communication'],
		description: 'Send SMS messages using the Twilio API',
		icon: 'https://upload.wikimedia.org/wikipedia/commons/c/cf/Twilio_logo.svg',
		url: 'https://www.twilio.com/console',
		form: {
			config: {
				accountSid: {
					description: 'Your Twilio Account SID',
					label: 'Account SID',
					required: true,
				},
				fromNumber: {
					description: 'The Twilio phone number to send messages from (in E.164 format)',
					label: 'From Number',
					required: true,
				},
			},
			secrets: {
				authToken: {
					description: 'Your Twilio Auth Token',
					label: 'Auth Token',
					required: true,
				},
			},
		},
	};
}
