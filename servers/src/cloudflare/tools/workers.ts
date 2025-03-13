import { Tool } from '@modelcontextprotocol/sdk/types.js';
import { ToolHandlers } from '../utils/types.js';

interface CloudflareWorkerListResponse {
	result: Array<{
		id: string;
		name: string;
		script?: string;
		modified_on?: string;
	}>;
	success: boolean;
	errors: any[];
	messages: any[];
}

// New Worker Tool definitions
const WORKER_LIST_TOOL: Tool = {
	name: 'worker_list',
	description: 'List all Workers in your account',
	inputSchema: {
		type: 'object',
		properties: {},
	},
};
const WORKER_GET_TOOL: Tool = {
	name: 'worker_get',
	description: "Get a Worker's script content",
	inputSchema: {
		type: 'object',
		properties: {
			name: {
				type: 'string',
				description: 'Name of the Worker script',
			},
		},
		required: ['name'],
	},
};

// Update the WORKER_PUT_TOOL definition
const WORKER_PUT_TOOL: Tool = {
	name: 'worker_put',
	description: 'Create or update a Worker script with optional bindings and compatibility settings',
	inputSchema: {
		type: 'object',
		properties: {
			name: {
				type: 'string',
				description: 'Name of the Worker script',
			},
			script: {
				type: 'string',
				description: 'The Worker script content',
			},
			bindings: {
				type: 'array',
				description: 'Optional array of resource bindings (KV, R2, D1, etc)',
				items: {
					type: 'object',
					properties: {
						type: {
							type: 'string',
							description: 'Type of binding (kv_namespace, r2_bucket, d1_database, service, analytics_engine, queue, durable_object)',
							enum: ['kv_namespace', 'r2_bucket', 'd1_database', 'service', 'analytics_engine', 'queue', 'durable_object_namespace'],
						},
						name: {
							type: 'string',
							description: 'Name of the binding in the Worker code',
						},
						namespace_id: {
							type: 'string',
							description: 'ID of the KV namespace (required for kv_namespace type)',
						},
						bucket_name: {
							type: 'string',
							description: 'Name of the R2 bucket (required for r2_bucket type)',
						},
						database_id: {
							type: 'string',
							description: 'ID of the D1 database (required for d1_database type)',
						},
						service: {
							type: 'string',
							description: 'Name of the service (required for service type)',
						},
						dataset: {
							type: 'string',
							description: 'Name of the analytics dataset (required for analytics_engine type)',
						},
						queue_name: {
							type: 'string',
							description: 'Name of the queue (required for queue type)',
						},
						class_name: {
							type: 'string',
							description: 'Name of the Durable Object class (required for durable_object_namespace type)',
						},
						script_name: {
							type: 'string',
							description: 'Optional script name for external Durable Object bindings',
						},
					},
					required: ['type', 'name'],
				},
			},
			migrations: {
				type: 'object',
				description:
					'Optional migrations object which describes the set of new/changed/deleted Durable Objects to apply when deploying this worker e.g. adding a new Durable Object for the first time requires an entry in the "new_sqlite_classes" or "new_classes" property.',
				items: {
					properties: {
						new_tag: {
							type: 'string',
							description: 'The current version after applying this migration (e.g., "v1", "v2")',
						},
						new_classes: {
							type: 'array',
							items: { type: 'string' },
							description: 'The new Durable Objects using legacy storage being added',
						},
						new_sqlite_classes: {
							type: 'array',
							items: { type: 'string' },
							description: 'The new Durable Objects using the new, improved SQLite storage being added',
						},
						renamed_classes: {
							type: 'array',
							items: {
								type: 'object',
								properties: {
									from: { type: 'string' },
									to: { type: 'string' },
								},
								required: ['from', 'to'],
							},
							description: 'The Durable Objects being renamed',
						},
						deleted_classes: {
							type: 'array',
							items: { type: 'string' },
							description: 'The Durable Objects being removed',
						},
					},
					required: ['tag'],
				},
			},
			compatibility_date: {
				type: 'string',
				description: 'Optional compatibility date for the Worker (e.g., "2024-01-01")',
			},
			compatibility_flags: {
				type: 'array',
				description: 'Optional array of compatibility flags',
				items: {
					type: 'string',
				},
			},
			skip_workers_dev: {
				type: 'boolean',
				description: `Do not deploy the Worker on your workers.dev subdomain. Should be set to true if the user already has a domain name, or doesn't want this worker to be publicly accessible..`,
			},
			no_observability: {
				type: 'boolean',
				description:
					'Disable Worker Logs for this worker, which automatically ingests logs emitted from Cloudflare Workers and lets you filter, and analyze them in the Cloudflare dashboard.',
			},
		},
		required: ['name', 'script'],
	},
};

const WORKER_DELETE_TOOL: Tool = {
	name: 'worker_delete',
	description: 'Delete a Worker script',
	inputSchema: {
		type: 'object',
		properties: {
			name: {
				type: 'string',
				description: 'Name of the Worker script',
			},
		},
		required: ['name'],
	},
};
export const WORKER_TOOLS = [WORKER_LIST_TOOL, WORKER_GET_TOOL, WORKER_PUT_TOOL, WORKER_DELETE_TOOL];

export async function handleWorkerList(accountId: string, apiToken: string): Promise<any> {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/workers/scripts`;

	const response = await fetch(url, {
		headers: { Authorization: `Bearer ${apiToken}` },
	});

	if (!response.ok) {
		const error = await response.text();
		throw new Error(`Failed to list workers: ${error}`);
	}

	const data = await response.json();
	return data.result;
}

export async function handleWorkerGet(accountId: string, apiToken: string, name: string) {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/workers/scripts/${name}`;

	const response = await fetch(url, {
		headers: { Authorization: `Bearer ${apiToken}` },
	});

	if (!response.ok) {
		const error = await response.text();

		throw new Error(`Failed to get worker: ${error}`);
	}

	const data = await response.text();
	return data;
}

export interface Observability {
	/** If observability is enabled for this Worker */
	enabled: boolean;
	/** The sampling rate */
	head_sampling_rate?: number;
}

interface CfDurableObjectMigrations {
	tag: string;
	new_classes?: string[];
	new_sqlite_classes?: string[];
	renamed_classes?: {
		from: string;
		to: string;
	}[];
	deleted_classes?: string[];
}

interface DurableObjectBinding {
	type: 'durable_object_namespace';
	name: string;
	class_name: string;
	script_name?: string; // Optional, defaults to the current worker
}

// Update WorkerBinding to include Durable Objects
type WorkerMetadataBinding =
	| {
			type: 'kv_namespace';
			name: string;
			namespace_id: string;
	  }
	| {
			type: 'r2_bucket';
			name: string;
			bucket_name: string;
	  }
	| {
			type: 'd1_database';
			name: string;
			database_id: string;
	  }
	| {
			type: 'service';
			name: string;
			service: string;
	  }
	| {
			type: 'analytics_engine';
			name: string;
			dataset: string;
	  }
	| {
			type: 'queue';
			name: string;
			queue_name: string;
	  }
	| DurableObjectBinding;

// Update the handleWorkerPut function
export async function handleWorkerPut(
	accountId: string,
	apiToken: string,
	name: string,
	script: string,
	bindings?: WorkerMetadataBinding[],
	compatibility_date?: string,
	compatibility_flags?: string[],
	migrations?: CfDurableObjectMigrations,
	workers_dev?: boolean,
	observability?: boolean
) {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/workers/scripts/${name}`;

	const metadata = {
		main_module: 'worker.js',
		bindings: bindings || [],
		compatibility_date: compatibility_date || '2024-01-01',
		compatibility_flags: compatibility_flags || [],
		...(migrations ? { migrations } : {}),
		observability: observability ? { enabled: true } : undefined,
	};

	// Create form data with metadata and script
	const formData = new FormData();
	formData.set('metadata', JSON.stringify(metadata));
	formData.set('worker.js', script);

	const response = await fetch(url, {
		method: 'PUT',
		headers: {
			Authorization: `Bearer ${apiToken}`,
		},
		body: formData,
	});

	if (!response.ok) {
		const error = await response.text();

		throw new Error(`Failed to put worker: ${error}`);
	}

	if (workers_dev) {
		const response = await fetch(url + '/subdomain', {
			method: 'POST',
			body: JSON.stringify({
				enabled: true,
			}),
			headers: {
				Authorization: `Bearer ${apiToken}`,
				'Content-Type': 'application/json',
			},
		});

		if (!response.ok) {
			const error = await response.text();

			throw new Error(`Failed to update subdomain: ${error}`);
		}
	}

	return 'Success';
}

export async function handleWorkerDelete(accountId: string, apiToken: string, name: string) {
	const url = `https://api.cloudflare.com/client/v4/accounts/${accountId}/workers/scripts/${name}`;

	const response = await fetch(url, {
		method: 'DELETE',
		headers: { Authorization: `Bearer ${apiToken}` },
	});

	if (!response.ok) {
		const error = await response.text();

		throw new Error(`Failed to delete worker: ${error}`);
	}

	return 'Success';
}

export const WORKERS_HANDLERS: ToolHandlers = {
	worker_list: async (request: { name: string; arguments: Record<string, any> }, accountId: string, apiToken: string) => {
		const results = await handleWorkerList(accountId, apiToken);
		return {
			content: [
				{
					type: 'text',
					text: JSON.stringify(results, null, 2),
				},
			],
		};
	},

	worker_get: async (request: { name: string; arguments: Record<string, any> }, accountId: string, apiToken: string) => {
		const { name } = request.arguments;
		const script = await handleWorkerGet(accountId, apiToken, name);
		return {
			content: [
				{
					type: 'text',
					text: script,
				},
			],
		};
	},

	worker_put: async (request: { name: string; arguments: Record<string, any> }, accountId: string, apiToken: string) => {
		const { name, script, bindings, compatibility_date, compatibility_flags, migrations, skip_workers_dev, no_observability } = request.arguments;
		await handleWorkerPut(
			accountId,
			apiToken,
			name,
			script,
			bindings,
			compatibility_date,
			compatibility_flags,
			migrations,
			!skip_workers_dev,
			!no_observability
		);
		return {
			content: [
				{
					type: 'text',
					text: `Successfully deployed worker: ${name}`,
				},
			],
		};
	},

	worker_delete: async (request: { name: string; arguments: Record<string, any> }, accountId: string, apiToken: string) => {
		const { name } = request.arguments;
		await handleWorkerDelete(accountId, apiToken, name);
		return {
			content: [
				{
					type: 'text',
					text: `Successfully deleted worker: ${name}`,
				},
			],
		};
	},
};
