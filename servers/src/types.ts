
export type ToolResponse = {
	content?: any;
	toolResult?: any;
	isError?: boolean;
};

export type ListToolsResponse = {
	tools: any[];
};

export interface MCPUpdate {
	type: string;
	data: any;
	metadata?: Record<string, any>;
}

export interface MCPServer {
	list: () => Promise<ListToolsResponse>;
	call: (request: Request, config: Record<string, string>, secrets: Record<string, string>) => Promise<ToolResponse>;
	getUpdates: () => Promise<MCPUpdate | MCPUpdate[] | null>;
	infos?: () => Promise<any>;
}
