import { GdriveClient } from '../client.js';
import { listFilesTool } from '../types.js';

export async function listFiles(gdriveClient: GdriveClient): Promise<any> {
	return await gdriveClient.listFiles();
}

export { listFilesTool };
