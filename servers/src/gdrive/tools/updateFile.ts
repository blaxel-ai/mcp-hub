import { GdriveClient } from '../client.js';
import { updateFileTool } from '../types.js';

export async function updateFile(gdriveClient: GdriveClient, fileId: string, content: string, name?: string): Promise<any> {
	if (!fileId || !content) {
		throw new Error('File ID and content are required');
	}
	return await gdriveClient.updateFile(fileId, content, name);
}

export { updateFileTool };
