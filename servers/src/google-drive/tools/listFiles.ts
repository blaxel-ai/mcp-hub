import { GoogleDriveClient } from '../client.js';
import { listFilesTool } from '../types.js';

export async function listFiles(googleDriveClient: GoogleDriveClient): Promise<any> {
	return await googleDriveClient.listFiles();
}

export { listFilesTool };
