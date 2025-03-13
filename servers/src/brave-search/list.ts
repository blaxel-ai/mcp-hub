import { ListToolsResponse } from '../types.js';
import { LOCAL_SEARCH_TOOL, WEB_SEARCH_TOOL } from './tools.js';

export async function list(): Promise<ListToolsResponse> {
	return { tools: [WEB_SEARCH_TOOL, LOCAL_SEARCH_TOOL] };
}
