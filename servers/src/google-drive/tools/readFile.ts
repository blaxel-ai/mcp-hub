import { GoogleDriveClient } from '../client.js';
import { readFileTool } from '../types.js';

export async function readFile(googleDriveClient: GoogleDriveClient, fileId: string): Promise<any> {
	if (!fileId) {
		throw new Error('File ID is required');
	}
	return await googleDriveClient.readFile(fileId);
}

export { readFileTool };
