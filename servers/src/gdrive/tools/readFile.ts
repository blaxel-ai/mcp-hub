import { GdriveClient } from '../client.js';
import { readFileTool } from '../types.js';

export async function readFile(gdriveClient: GdriveClient, fileId: string): Promise<any> {
	if (!fileId) {
		throw new Error('File ID is required');
	}
	return await gdriveClient.readFile(fileId);
}

export { readFileTool };
