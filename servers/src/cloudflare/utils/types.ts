import { Result } from '@modelcontextprotocol/sdk/types.js';

export type ToolHandlers = Record<
	string,
	(request: { name: string; arguments: Record<string, any> }, accountId: string, apiToken: string) => Promise<Result>
>;
