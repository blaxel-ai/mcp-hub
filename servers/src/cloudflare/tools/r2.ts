// Add R2 tool definitions
import { Tool } from '@modelcontextprotocol/sdk/types.js';
import { ToolHandlers } from '../utils/types.js';

const R2_LIST_BUCKETS_TOOL: Tool = {
	name: 'r2_list_buckets',
	description: 'List all R2 buckets in your account',
	inputSchema: {
		type: 'object',
		properties: {},
	},
};
const R2_CREATE_BUCKET_TOOL: Tool = {
	name: 'r2_create_bucket',
	description: 'Create a new R2 bucket',
	inputSchema: {
		type: 'object',
		properties: {
			name: {
				type: 'string',
				description: 'Name of the bucket to create',
			},
		},
		required: ['name'],
	},
};
const R2_DELETE_BUCKET_TOOL: Tool = {
	name: 'r2_delete_bucket',
	description: 'Delete an R2 bucket',
	inputSchema: {
		type: 'object',
		properties: {
			name: {
				type: 'string',
				description: 'Name of the bucket to delete',
			},
		},
		required: ['name'],
	},
};
const R2_LIST_OBJECTS_TOOL: Tool = {
	name: 'r2_list_objects',
	description: 'List objects in an R2 bucket',
	inputSchema: {
		type: 'object',
		properties: {
			bucket: {
				type: 'string',
				description: 'Name of the bucket',
			},
			prefix: {
				type: 'string',
				description: 'Optional prefix to filter objects',
			},
			delimiter: {
				type: 'string',
				description: 'Optional delimiter for hierarchical listing',
			},
			limit: {
				type: 'number',
				description: 'Maximum number of objects to return',
			},
		},
		required: ['bucket'],
	},
};
const R2_GET_OBJECT_TOOL: Tool = {
	name: 'r2_get_object',
	description: 'Get an object from an R2 bucket',
	inputSchema: {
		type: 'object',
		properties: {
			bucket: {
				type: 'string',
				description: 'Name of the bucket',
			},
			key: {
				type: 'string',
				description: 'Key of the object to get',
			},
		},
		required: ['bucket', 'key'],
	},
};
const R2_PUT_OBJECT_TOOL: Tool = {
	name: 'r2_put_object',
	description: 'Put an object into an R2 bucket',
	inputSchema: {
		type: 'object',
		properties: {
			bucket: {
				type: 'string',
				description: 'Name of the bucket',
			},
			key: {
				type: 'string',
				description: 'Key of the object to put',
			},
			content: {
				type: 'string',
				description: 'Content to store in the object',
			},
			contentType: {
				type: 'string',
				description: 'Optional MIME type of the content',
			},
		},
		required: ['bucket', 'key', 'content'],
	},
};
const R2_DELETE_OBJECT_TOOL: Tool = {
	name: 'r2_delete_object',
	description: 'Delete an object from an R2 bucket',
	inputSchema: {
		type: 'object',
		properties: {
			bucket: {
				type: 'string',
				description: 'Name of the bucket',
			},
			key: {
				type: 'string',
				description: 'Key of the object to delete',
			},
		},
		required: ['bucket', 'key'],
	},
};
export const R2_TOOLS = [
	R2_LIST_BUCKETS_TOOL,
	R2_CREATE_BUCKET_TOOL,
	R2_DELETE_BUCKET_TOOL,
	R2_LIST_OBJECTS_TOOL,
	R2_GET_OBJECT_TOOL,
	R2_PUT_OBJECT_TOOL,
	R2_DELETE_OBJECT_TOOL,
];

// Add R2 response interfaces
interface CloudflareR2BucketsResponse {
	result: Array<{
		name: string;
		creation_date: string;
	}>;
	success: boolean;
	errors: any[];
	messages: any[];
}

interface CloudflareR2ObjectsResponse {
	objects: Array<{
		key: string;
		size: number;
		uploaded: string;
		etag: string;
		httpEtag: string;
		version: string;
	}>;
	delimitedPrefixes: string[];
	truncated: boolean;
}

// Add R2 handlers
export async function handleR2ListBuckets(accountId: string, apiToken: string) {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/r2/buckets`;

	const response = await fetch(url, {
		headers: { Authorization: `Bearer ${apiToken}` },
	});

	if (!response.ok) {
		const error = await response.text();
		throw new Error(`Failed to list R2 buckets: ${error}`);
	}

	const data = (await response.json()) as CloudflareR2BucketsResponse;
	return data.result;
}

export async function handleR2CreateBucket(accountId: string, apiToken: string, name: string) {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/r2/buckets`;

	const response = await fetch(url, {
		method: 'POST',
		headers: {
			Authorization: `Bearer ${apiToken}`,
			'Content-Type': 'application/json',
		},
		body: JSON.stringify({ name }),
	});

	if (!response.ok) {
		const error = await response.text();
		throw new Error(`Failed to create R2 bucket: ${error}`);
	}

	return 'Success';
}

export async function handleR2DeleteBucket(accountId: string, apiToken: string, name: string) {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/r2/buckets/${name}`;

	const response = await fetch(url, {
		method: 'DELETE',
		headers: { Authorization: `Bearer ${apiToken}` },
	});

	if (!response.ok) {
		const error = await response.text();
		throw new Error(`Failed to delete R2 bucket: ${error}`);
	}

	return 'Success';
}

export async function handleR2ListObjects(
	accountId: string,
	apiToken: string,
	bucket: string,
	prefix?: string,
	delimiter?: string,
	limit?: number
) {
	const params = new URLSearchParams();
	if (prefix) params.append('prefix', prefix);
	if (delimiter) params.append('delimiter', delimiter);
	if (limit) params.append('limit', limit.toString());

	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/r2/buckets/${bucket}/objects?${params}`;

	const response = await fetch(url, {
		headers: { Authorization: `Bearer ${apiToken}` },
	});

	if (!response.ok) {
		const error = await response.text();
		throw new Error(`Failed to list R2 objects: ${error}`);
	}

	const data = (await response.json()) as CloudflareR2ObjectsResponse;
	return data;
}

export async function handleR2GetObject(accountId: string, apiToken: string, bucket: string, key: string) {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/r2/buckets/${bucket}/objects/${key}`;

	const response = await fetch(url, {
		headers: { Authorization: `Bearer ${apiToken}` },
	});

	if (!response.ok) {
		const error = await response.text();
		throw new Error(`Failed to get R2 object: ${error}`);
	}

	const content = await response.text();
	return content;
}

export async function handleR2PutObject(
	accountId: string,
	apiToken: string,
	bucket: string,
	key: string,
	content: string,
	contentType?: string
) {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/r2/buckets/${bucket}/objects/${key}`;

	const headers: Record<string, string> = {
		Authorization: `Bearer ${apiToken}`,
	};
	if (contentType) {
		headers['Content-Type'] = contentType;
	}

	const response = await fetch(url, {
		method: 'PUT',
		headers,
		body: content,
	});

	if (!response.ok) {
		const error = await response.text();
		throw new Error(`Failed to put R2 object: ${error}`);
	}

	return 'Success';
}

export async function handleR2DeleteObject(accountId: string, apiToken: string, bucket: string, key: string) {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/r2/buckets/${bucket}/objects/${key}`;

	const response = await fetch(url, {
		method: 'DELETE',
		headers: { Authorization: `Bearer ${apiToken}` },
	});

	if (!response.ok) {
		const error = await response.text();
		throw new Error(`Failed to delete R2 object: ${error}`);
	}

	return 'Success';
}

export const R2_HANDLERS: ToolHandlers = {
	// Add R2 cases to the tool call handler
	r2_list_buckets: async (request, accountId, apiToken) => {
		const results = await handleR2ListBuckets(accountId, apiToken);
		return {
			content: [{ type: 'text', text: JSON.stringify(results, null, 2) }],
		};
	},

	r2_create_bucket: async (request, accountId, apiToken) => {
		const { name } = request as { name: string };
		await handleR2CreateBucket(accountId, apiToken, name);
		return {
			content: [{ type: 'text', text: `Successfully created bucket: ${name}` }],
		};
	},

	r2_delete_bucket: async (request, accountId, apiToken) => {
		const { name } = request as { name: string };
		await handleR2DeleteBucket(accountId, apiToken, name);
		return {
			content: [{ type: 'text', text: `Successfully deleted bucket: ${name}` }],
		};
	},

	r2_list_objects: async (request, accountId, apiToken) => {
		const { bucket, prefix, delimiter, limit } = request.arguments as {
			bucket: string;
			prefix?: string;
			delimiter?: string;
			limit?: number;
		};
		const results = await handleR2ListObjects(accountId, apiToken, bucket, prefix, delimiter, limit);
		return {
			content: [{ type: 'text', text: JSON.stringify(results, null, 2) }],
		};
	},

	r2_get_object: async (request: { name: string; arguments: Record<string, any> }, accountId: string, apiToken: string) => {
		const { bucket, key } = request.arguments;
		const content = await handleR2GetObject(accountId, apiToken, bucket, key);
		return {
			content: [{ type: 'text', text: content }],
		};
	},

	r2_put_object: async (request, accountId, apiToken) => {
		const { bucket, key, content, contentType } = request.arguments as {
			bucket: string;
			key: string;
			content: string;
			contentType?: string;
		};
		await handleR2PutObject(accountId, apiToken, bucket, key, content, contentType);
		return {
			content: [{ type: 'text', text: `Successfully stored object: ${key}` }],
		};
	},

	r2_delete_object: async (request, accountId, apiToken) => {
		const { bucket, key } = request.arguments as { bucket: string; key: string };
		await handleR2DeleteObject(accountId, apiToken, bucket, key);
		return {
			content: [{ type: 'text', text: `Successfully deleted object: ${key}` }],
		};
	},
};
