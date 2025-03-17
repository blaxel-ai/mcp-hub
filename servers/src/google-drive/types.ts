import { Tool } from '@modelcontextprotocol/sdk/types.js';

// Tool definitions
export const listFilesTool: Tool = {
	name: 'list_files',
	description: `
List files in your Google Drive.
`,
	inputSchema: {
		type: 'object',
		properties: {},
	},
};

export const readFileTool: Tool = {
	name: 'read_file',
	description: `
Read a file from your Google Drive by its ID.
`,
	inputSchema: {
		type: 'object',
		properties: {
			fileId: {
				type: 'string',
				description: 'The ID of the file to read',
			},
		},
		required: ['fileId'],
	},
};

export const createFileTool: Tool = {
	name: 'create_file',
	description: `
Create a new file in your Google Drive.
`,
	inputSchema: {
		type: 'object',
		properties: {
			name: {
				type: 'string',
				description: 'The name of the file to create',
			},
			content: {
				type: 'string',
				description: 'The content of the file',
			},
			mimeType: {
				type: 'string',
				description: 'The MIME type of the file (default: text/plain)',
			},
			folderId: {
				type: 'string',
				description: 'The ID of the folder to create the file in (optional)',
			},
		},
		required: ['name', 'content'],
	},
};

export const updateFileTool: Tool = {
	name: 'update_file',
	description: `
Update an existing file in your Google Drive by its ID.
`,
	inputSchema: {
		type: 'object',
		properties: {
			fileId: {
				type: 'string',
				description: 'The ID of the file to update',
			},
			content: {
				type: 'string',
				description: 'The new content of the file',
			},
			name: {
				type: 'string',
				description: 'The new name of the file (optional)',
			},
		},
		required: ['fileId', 'content'],
	},
};
