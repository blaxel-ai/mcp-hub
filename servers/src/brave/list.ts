import { ListToolsResponse } from '../types';
import { LOCAL_SEARCH_TOOL, WEB_SEARCH_TOOL } from './tools';

export async function list(): Promise<ListToolsResponse> {
	return { tools: [WEB_SEARCH_TOOL, LOCAL_SEARCH_TOOL] };
}
