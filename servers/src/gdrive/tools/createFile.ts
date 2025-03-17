import { GdriveClient } from '../client.js';
import { createFileTool } from '../types.js';

export async function createFile(
	gdriveClient: GdriveClient,
	name: string,
	content: string,
	mimeType: string = 'text/plain',
	folderId?: string
): Promise<any> {
	if (!name || !content) {
		throw new Error('File name and content are required');
	}
	return await gdriveClient.createFile(name, content, mimeType, folderId);
}

export { createFileTool };
