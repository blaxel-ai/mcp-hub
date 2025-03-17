#!/usr/bin/env node
import { GoogleDriveClient } from './client.js';
import { createFile, createFileTool, listFiles, listFilesTool, readFile, readFileTool, updateFile, updateFileTool } from './tools/index.js';

export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	try {
		const googleDriveClient = new GoogleDriveClient(config, secrets);
		const requestBody: { name: string; arguments: any } = (await request.json()) as { name: string; arguments: any };
		if (!requestBody.arguments) {
			throw new Error('No arguments provided');
		}

		switch (requestBody.name) {
			case 'list_files': {
				return await listFiles(googleDriveClient);
			}
			case 'read_file': {
				const { fileId } = requestBody.arguments;
				return await readFile(googleDriveClient, fileId);
			}
			case 'create_file': {
				const { name, content, mimeType = 'text/plain', folderId } = requestBody.arguments;
				return await createFile(googleDriveClient, name, content, mimeType, folderId);
			}
			case 'update_file': {
				const { fileId, content, name } = requestBody.arguments;
				return await updateFile(googleDriveClient, fileId, content, name);
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
		tools: [listFilesTool, readFileTool, createFileTool, updateFileTool],
	};
}

export async function infos() {
	return {
		name: 'google-drive',
		form: {
			oauth: {
				type: 'google',
				scope: ['https://www.googleapis.com/auth/drive.file'],
			},
			config: {},
			secrets: {
				clientId: {
					description: 'Google API client ID',
					label: 'Google API client ID',
					required: true,
					hidden: true,
				},
				clientSecret: {
					description: 'Google API client secret',
					label: 'Google API client secret',
					required: true,
					hidden: true,
				},
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
