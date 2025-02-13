#!/usr/bin/env node
import { Tool } from '@modelcontextprotocol/sdk/types.js';
import postgres from 'postgres';

interface QueryArgs {
	sql: string;
}

// Tool definitions
const queryTool: Tool = {
	name: 'query',
	description:
		'Execute a read-only SQL query on the PostgreSQL database. Use this to retrieve data with SELECT statements. The query must not modify any data.',
	inputSchema: {
		type: 'object',
		properties: {
			sql: {
				type: 'string',
				description: 'The SQL query to execute. Must be a SELECT statement or other read-only operation.',
			},
		},
		required: ['sql'],
	},
};

const listTablesTool: Tool = {
	name: 'list_tables',
	description:
		'Retrieve a list of all available tables in the public schema of the PostgreSQL database. This helps explore the database structure.',
	inputSchema: {
		type: 'object',
		properties: {},
	},
};

class PostgresClient {
	private client: postgres.Sql;

	constructor(databaseUrl: string, password: string) {
		const resourceBaseUrl = new URL(databaseUrl);
		resourceBaseUrl.protocol = 'postgres:';
		resourceBaseUrl.password = password;

		const sql = postgres(resourceBaseUrl.toString());
		this.client = sql;
	}

	async query(request: string): Promise<any> {
		try {
			const result = await this.client.begin(async (sql) => {
				await sql`SET TRANSACTION READ ONLY`;
				return await sql.unsafe(request);
			});
			return {
				content: [{ type: 'text', text: JSON.stringify(result, null, 2) }],
				isError: false,
			};
		} catch (error) {
			return {
				content: [{ type: 'text', text: JSON.stringify(error, null, 2) }],
				isError: true,
			};
		}
	}

	async listTables(): Promise<any> {
		try {
			const result = await this.client.unsafe("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'");
			return {
				content: [
					{
						type: 'text',
						text: JSON.stringify(
							result.map((r) => r.table_name),
							null,
							2,
						),
					},
				],
				isError: false,
			};
		} catch (error) {
			return {
				content: [{ type: 'text', text: JSON.stringify(error, null, 2) }],
				isError: true,
			};
		}
	}
}

export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	try {
		const postgresClient = new PostgresClient(config.connectionString, secrets.password);
		const requestBody: { name: string; arguments: any } = await request.json() as { name: string; arguments: any };
		if (!requestBody.arguments) {
			throw new Error('No arguments provided');
		}

		switch (requestBody.name) {
			case 'query': {
				const args = requestBody.arguments as QueryArgs;
				return await postgresClient.query(args.sql);
			}
			case 'list_tables': {
				return await postgresClient.listTables();
			}

			default:
				throw new Error(`Unknown tool: ${requestBody.name}`);
		}
	} catch (error) {
		return {
			content: [
				{
					type: 'text',
					text: JSON.stringify({
						error: error instanceof Error ? error.message : String(error),
					}),
				},
			],
		};
	}
}

export async function list() {
	return {
		tools: [queryTool, listTablesTool],
	};
}

export async function infos() {
	return {
		name: 'postgres',
		displayName: 'PostgreSQL',
		integration: 'postgres',
		categories: ['database'],
		description: 'List tables and run queries in your PostgreSQL database',
		icon: 'https://www.postgresql.org/media/img/about/press/elephant.png',
		url: 'https://www.postgresql.org',
		form: {
			config: {
				connectionString: {
					description: 'connection string with format postgresql://postgres@localhost:5432/postgres',
					label: 'Database URL',
					required: true,
				},
			},
			secrets: {
				password: {
					description: 'Database password',
					label: 'Database password',
					required: false,
				},
			},
		},
	};
}
