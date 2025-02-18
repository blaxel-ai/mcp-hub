#!/usr/bin/env node
import { GetObjectCommand, ListObjectsV2Command, PutObjectCommand, S3Client } from '@aws-sdk/client-s3';
import { Tool } from '@modelcontextprotocol/sdk/types.js';
import mime from 'mime';

interface ListFilesArgs {
	bucket: string;
	prefix: string;
}

// Tool definitions
const listFilesTool: Tool = {
	name: 'list_files',
	description:
		'List all files in an S3 bucket with an optional prefix filter. Returns the objects metadata including names, sizes, and last modified dates.',
	inputSchema: {
		type: 'object',
		properties: {
			bucket: {
				type: 'string',
				description: 'The name of the S3 bucket to list files from',
			},
			prefix: {
				type: 'string',
				description: 'Optional prefix to filter files (like a folder path). Example: "documents/" or "images/2024/"',
			},
		},
		required: ['bucket'],
	},
};

interface RetrieveFileArgs {
	bucket: string;
	key: string;
}

const retrieveFileTool: Tool = {
	name: 'retrieve_file',
	description:
		'Download and retrieve the contents of a specific file from an S3 bucket. Text files will be returned as text, while binary files will be returned as base64-encoded data.',
	inputSchema: {
		type: 'object',
		properties: {
			bucket: {
				type: 'string',
				description: 'The name of the S3 bucket containing the file',
			},
			key: {
				type: 'string',
				description: 'The complete path and filename of the object in the bucket. Example: "folder/document.pdf"',
			},
		},
		required: ['bucket', 'key'],
	},
};

interface UploadFileArgs {
	bucket: string;
	key: string;
	data: string;
}

const uploadFileTool: Tool = {
	name: 'upload_file',
	description:
		'Upload a file to an S3 bucket. For binary files, the data should be base64-encoded. For text files, plain text content can be provided directly.',
	inputSchema: {
		type: 'object',
		properties: {
			bucket: {
				type: 'string',
				description: 'The name of the S3 bucket where the file will be uploaded',
			},
			key: {
				type: 'string',
				description: 'The desired path and filename in the bucket. Example: "uploads/document.pdf" or "images/photo.jpg"',
			},
			data: {
				type: 'string',
				description: 'The file content: plain text for text files, or base64-encoded string for binary files',
			},
		},
		required: ['bucket', 'key', 'data'],
	},
};

class AWSS3Client {
	private client: S3Client;

	constructor(config: Record<string, string>, secrets: Record<string, string>) {
		this.client = new S3Client({
			region: config.region,
			credentials: {
				accessKeyId: secrets.accessKeyId,
				secretAccessKey: secrets.secretAccessKey,
				sessionToken: secrets.sessionToken,
			},
		});
	}

	async listFiles(request: ListFilesArgs): Promise<any> {
		try {
			const params = {
				Bucket: request.bucket,
				Prefix: request.prefix,
			};
			const command = new ListObjectsV2Command(params);
			const result = await this.client.send(command);
			return {
				content: [
					{
						type: 'text',
						text: JSON.stringify(result.Contents?.map((f) => f.Key)),
					},
				],
				isError: false,
			};
		} catch (error) {
			return {
				content: [{ type: 'text', text: JSON.stringify(error) }],
				isError: true,
			};
		}
	}

	async retrieveFile(request: RetrieveFileArgs): Promise<any> {
		try {
			const params = {
				Bucket: request.bucket,
				Key: request.key,
			};
			const command = new GetObjectCommand(params);
			const result = await this.client.send(command);
			const mimetype = result.ContentType;

			// Read the stream into a buffer
			const streamToBuffer = async (stream: ReadableStream): Promise<Buffer> => {
				const chunks: Uint8Array[] = [];
				for await (const chunk of stream) {
					chunks.push(chunk);
				}
				return Buffer.concat(chunks);
			};

			const bodyBuffer = await streamToBuffer(result.Body as ReadableStream);

			// Handle text-based MIME types
			const textBasedTypes = [
				'text/plain',
				'text/html',
				'text/css',
				'text/csv',
				'text/javascript',
				'text/markdown',
				'text/xml',
				'text/yaml',
				'text/calendar',
				'text/x-python',
				'text/x-java-source',
				'text/x-c',
				'text/x-script.ruby',
				'application/json',
				'application/xml',
				'application/javascript',
				'application/typescript',
				'application/x-yaml',
				'application/ld+json',
				'application/graphql',
				'application/x-httpd-php',
			];

			if (mimetype && textBasedTypes.includes(mimetype)) {
				try {
					const textContent = bodyBuffer.toString('utf-8');
					return {
						content: [
							{
								type: 'text',
								text: JSON.stringify({
									text: textContent,
									name: request.key,
									mimeType: mimetype,
								}),
							},
						],
						isError: false,
					};
				} catch (conversionError: any) {
					// Fallback to binary if text conversion fails
					console.warn(`Failed to convert ${mimetype} file to text: ${conversionError.message}`);
				}
			}

			// For non-text files or if text conversion failed, return as binary
			return {
				content: [
					{
						type: 'text',
						text: JSON.stringify({
							name: request.key,
							mimeType: mimetype,
							blob: bodyBuffer.toString('base64'),
						}),
					},
				],
				isError: false,
			};
		} catch (error) {
			return {
				content: [
					{
						type: 'text',
						text: JSON.stringify(error),
					},
				],
				isError: true,
			};
		}
	}

	async uploadFile(request: UploadFileArgs): Promise<any> {
		try {
			const mimeType = mime.getType(request.key) || 'application/octet-stream';

			// Determine if the content is base64 encoded
			const isBase64 = /^[A-Za-z0-9+/]*={0,2}$/.test(request.data);

			// Convert base64 to Buffer for binary files
			const body = isBase64 ? Buffer.from(request.data, 'base64') : request.data;

			const params = {
				Bucket: request.bucket,
				Key: request.key,
				Body: body,
				ContentType: mimeType,
			};

			const command = new PutObjectCommand(params);
			await this.client.send(command);
			return {
				content: [
					{
						type: 'text',
						text: `File uploaded successfully to ${request.key}`,
					},
				],
				isError: false,
			};
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
				isError: true,
			};
		}
	}
}

export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	try {
		const awsS3Client = new AWSS3Client(config, secrets);
		const requestBody: { name: string; arguments: any } = (await request.json()) as { name: string; arguments: any };
		if (!requestBody.arguments) {
			throw new Error('No arguments provided');
		}

		switch (requestBody.name) {
			case 'list_files': {
				const args = requestBody.arguments as ListFilesArgs;
				return await awsS3Client.listFiles(args);
			}
			case 'retrieve_file': {
				const args = requestBody.arguments as RetrieveFileArgs;
				return await awsS3Client.retrieveFile(args);
			}
			case 'upload_file': {
				const args = requestBody.arguments as UploadFileArgs;
				return await awsS3Client.uploadFile(args);
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
		tools: [listFilesTool, retrieveFileTool, uploadFileTool],
	};
}

export async function infos() {
	return {
		name: 'aws-s3',
		displayName: 'AWS S3',
		integration: 'aws-s3',
		categories: ['storage'],
		description: 'Create, read and update objects in your S3 storage',
		icon: 'https://a.b.cdn.console.awsstatic.com/a/v1/DKY2SIL5N3MJQCULDNOQE7TKLNQIUXRSOHBJKJGQAHLZO7TLH3TQ/icon/c0828e0381730befd1f7a025057c74fb-43acc0496e64afba82dbc9ab774dc622.svg',
		url: 'https://aws.amazon.com/s3',
		form: {
			config: {
				region: {
					description: 'AWS region',
					label: 'AWS region',
					required: true,
				},
			},
			secrets: {
				accessKeyId: {
					description: 'AWS access key ID',
					label: 'AWS access key ID',
					required: true,
				},
				secretAccessKey: {
					description: 'AWS secret access key',
					label: 'AWS secret access key',
					required: true,
				},
				sessionToken: {
					description: 'AWS session token',
					label: 'AWS session token',
					required: false,
				},
			},
		},
	};
}
