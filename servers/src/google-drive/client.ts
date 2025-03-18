#!/usr/bin/env node

export class GoogleDriveClient {
	private refreshToken: string;
	private accessToken?: string;
	private clientId: string;
	private clientSecret: string;

	constructor(config: Record<string, string>, secrets: Record<string, string>) {
		this.refreshToken = secrets.refreshToken;
		this.clientId = secrets.clientId;
		this.clientSecret = secrets.clientSecret;
	}

	private async login() {
		if (this.accessToken) {
			return;
		}

		const response = await fetch('https://oauth2.googleapis.com/token', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
			},
			body: new URLSearchParams({
				client_id: this.clientId,
				client_secret: this.clientSecret,
				refresh_token: this.refreshToken,
				grant_type: 'refresh_token',
			}).toString(),
		});

		if (!response.ok) {
			const error = await response.json();
			throw new Error(`Failed to refresh token: ${JSON.stringify(error)}`);
		}

		const data: { access_token: string; id_token: string } = (await response.json()) as { access_token: string; id_token: string };
		this.accessToken = data.access_token;
	}

	async listFiles(): Promise<any> {
		await this.login();

		const response = await fetch('https://www.googleapis.com/drive/v3/files', {
			method: 'GET',
			headers: {
				Authorization: `Bearer ${this.accessToken}`,
				'Content-Type': 'application/json',
			},
		});
		if (!response.ok) {
			const error = await response.json();
			throw new Error(`Failed to list files: ${JSON.stringify(error)}`);
		}
		const data = await response.json();
		return {
			content: [{ type: 'text', text: JSON.stringify(data) }],
			isError: false,
		};
	}

	async readFile(fileId: string): Promise<any> {
		await this.login();

		// First, get the file metadata to determine if it's a Google Doc or a regular file
		const metadataResponse = await fetch(`https://www.googleapis.com/drive/v3/files/${fileId}?fields=mimeType,name`, {
			method: 'GET',
			headers: {
				Authorization: `Bearer ${this.accessToken}`,
				'Content-Type': 'application/json',
			},
		});

		if (!metadataResponse.ok) {
			const error = await metadataResponse.json();
			throw new Error(`Failed to get file metadata: ${JSON.stringify(error)}`);
		}

		const metadata = await metadataResponse.json();

		// Get the file content
		let contentUrl = `https://www.googleapis.com/drive/v3/files/${fileId}?alt=media`;

		// For Google Docs, Sheets, etc., we need to export them
		if (metadata.mimeType.startsWith('application/vnd.google-apps.')) {
			// Map Google Docs types to export formats
			let exportFormat = 'text/plain';
			if (metadata.mimeType === 'application/vnd.google-apps.document') {
				exportFormat = 'application/vnd.openxmlformats-officedocument.wordprocessingml.document';
			} else if (metadata.mimeType === 'application/vnd.google-apps.spreadsheet') {
				exportFormat = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';
			} else if (metadata.mimeType === 'application/vnd.google-apps.presentation') {
				exportFormat = 'application/vnd.openxmlformats-officedocument.presentationml.presentation';
			}
			contentUrl = `https://www.googleapis.com/drive/v3/files/${fileId}/export?mimeType=${encodeURIComponent(exportFormat)}`;
		}

		const contentResponse = await fetch(contentUrl, {
			method: 'GET',
			headers: {
				Authorization: `Bearer ${this.accessToken}`,
			},
		});

		if (!contentResponse.ok) {
			const error = await contentResponse.text();
			throw new Error(`Failed to read file content: ${error}`);
		}

		// Try to get content as text, but handle binary files
		let content;
		try {
			content = await contentResponse.text();
		} catch (e) {
			content = '[Binary content - cannot display]';
		}

		return {
			content: [
				{
					type: 'text',
					text: JSON.stringify({
						name: metadata.name,
						mimeType: metadata.mimeType,
						content: content,
					}),
				},
			],
			isError: false,
		};
	}

	async createFile(name: string, content: string, mimeType: string = 'text/plain', folderId?: string): Promise<any> {
		await this.login();

		// Create file metadata
		const metadata: any = {
			name,
			mimeType,
		};

		// If folder ID is provided, set the parent
		if (folderId) {
			metadata.parents = [folderId];
		}

		// For multipart upload
		const boundary = 'boundary' + Math.random().toString().substr(2);

		// Create multipart request body
		let requestBody = `--${boundary}\r\n`;
		requestBody += 'Content-Type: application/json; charset=UTF-8\r\n\r\n';
		requestBody += JSON.stringify(metadata) + '\r\n';
		requestBody += `--${boundary}\r\n`;
		requestBody += `Content-Type: ${mimeType}\r\n\r\n`;
		requestBody += content + '\r\n';
		requestBody += `--${boundary}--`;

		// Upload the file
		const response = await fetch('https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart', {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${this.accessToken}`,
				'Content-Type': `multipart/related; boundary=${boundary}`,
			},
			body: requestBody,
		});

		if (!response.ok) {
			const error = await response.json();
			throw new Error(`Failed to create file: ${JSON.stringify(error)}`);
		}

		const data = await response.json();
		return {
			content: [{ type: 'text', text: JSON.stringify(data) }],
			isError: false,
		};
	}

	async updateFile(fileId: string, content: string, name?: string): Promise<any> {
		await this.login();

		// First, get the file metadata to determine its MIME type
		const metadataResponse = await fetch(`https://www.googleapis.com/drive/v3/files/${fileId}?fields=mimeType`, {
			method: 'GET',
			headers: {
				Authorization: `Bearer ${this.accessToken}`,
				'Content-Type': 'application/json',
			},
		});

		if (!metadataResponse.ok) {
			const error = await metadataResponse.json();
			throw new Error(`Failed to get file metadata: ${JSON.stringify(error)}`);
		}

		const metadata = await metadataResponse.json();
		const mimeType = metadata.mimeType;

		// Create metadata for update
		const updateMetadata: any = {};
		if (name) {
			updateMetadata.name = name;
		}

		// For multipart upload
		const boundary = 'boundary' + Math.random().toString().substr(2);

		// Create multipart request body
		let requestBody = `--${boundary}\r\n`;
		requestBody += 'Content-Type: application/json; charset=UTF-8\r\n\r\n';
		requestBody += JSON.stringify(updateMetadata) + '\r\n';
		requestBody += `--${boundary}\r\n`;
		requestBody += `Content-Type: ${mimeType}\r\n\r\n`;
		requestBody += content + '\r\n';
		requestBody += `--${boundary}--`;

		// Update the file
		const response = await fetch(`https://www.googleapis.com/upload/drive/v3/files/${fileId}?uploadType=multipart`, {
			method: 'PATCH',
			headers: {
				Authorization: `Bearer ${this.accessToken}`,
				'Content-Type': `multipart/related; boundary=${boundary}`,
			},
			body: requestBody,
		});

		if (!response.ok) {
			const error = await response.json();
			throw new Error(`Failed to update file: ${JSON.stringify(error)}`);
		}

		const data = await response.json();
		return {
			content: [{ type: 'text', text: JSON.stringify(data) }],
			isError: false,
		};
	}
}
