import { GoogleDriveClient } from '../client.js';
import { updateFileTool } from '../types.js';

export async function updateFile(googleDriveClient: GoogleDriveClient, fileId: string, content: string, name?: string): Promise<any> {
	if (!fileId || !content) {
		throw new Error('File ID and content are required');
	}
	return await googleDriveClient.updateFile(fileId, content, name);
}

export { updateFileTool };
