import { z } from 'zod';
import { GitHubReferenceSchema } from '../common/types';
import { githubRequest } from '../common/utils';

// Schema definitions
export const CreateBranchOptionsSchema = z.object({
	ref: z.string(),
	sha: z.string().optional().describe('SHA of the commit to update the branch to'),
});

export const CreateBranchSchema = z.object({
	owner: z.string().describe('Repository owner (username or organization)'),
	repo: z.string().describe('Repository name'),
	branch: z.string().describe('Name for the new branch'),
	options: CreateBranchOptionsSchema.optional().describe('Options for creating the branch'),
});

export const CreateBranchFromRefSchema = z.object({
	owner: z.string().describe('Repository owner (username or organization)'),
	repo: z.string().describe('Repository name'),
	branch: z.string().describe('Name for the new branch'),
	from_branch: z.string().describe('Source branch to create from'),
});

export const UpdateBranchSchema = z.object({
	owner: z.string().describe('Repository owner (username or organization)'),
	repo: z.string().describe('Repository name'),
	branch: z.string().describe('Name for the new branch'),
	sha: z.string().describe('SHA of the commit to update the branch to'),
});

export const GetBranchSHA = z.object({
	owner: z.string().describe('Repository owner (username or organization)'),
	repo: z.string().describe('Repository name'),
	branch: z.string().describe('Name for the new branch'),
});

// Type exports
export type CreateBranchOptions = z.infer<typeof CreateBranchOptionsSchema>;
export type CreateBranchFromRefOptions = z.infer<typeof CreateBranchFromRefSchema>;
export type UpdateBranchOptions = z.infer<typeof UpdateBranchSchema>;
export type GetBranchSHAOptions = z.infer<typeof GetBranchSHA>;

// Function implementations
export async function getDefaultBranchSHA(secrets: Record<string, string>, owner: string, repo: string): Promise<string> {
	try {
		const response = await githubRequest(`https://api.github.com/repos/${owner}/${repo}/git/refs/heads/main`, secrets);
		const data = GitHubReferenceSchema.parse(response);
		return data.object.sha;
	} catch (error) {
		const masterResponse = await githubRequest(`https://api.github.com/repos/${owner}/${repo}/git/refs/heads/master`, secrets);
		if (!masterResponse) {
			throw new Error("Could not find default branch (tried 'main' and 'master')");
		}
		const data = GitHubReferenceSchema.parse(masterResponse);
		return data.object.sha;
	}
}

export async function createBranch(
	secrets: Record<string, string>,
	owner: string,
	repo: string,
	options: CreateBranchOptions,
): Promise<z.infer<typeof GitHubReferenceSchema>> {
	const fullRef = `refs/heads/${options.ref}`;

	const response = await githubRequest(`https://api.github.com/repos/${owner}/${repo}/git/refs`, secrets, {
		method: 'POST',
		body: {
			ref: fullRef,
			sha: options.sha,
		},
	});

	return GitHubReferenceSchema.parse(response);
}

export async function getBranchSHA(secrets: Record<string, string>, owner: string, repo: string, branch: string): Promise<string> {
	const response = await githubRequest(`https://api.github.com/repos/${owner}/${repo}/git/refs/heads/${branch}`, secrets);

	const data = GitHubReferenceSchema.parse(response);
	return data.object.sha;
}

export async function createBranchFromRef(
	secrets: Record<string, string>,
	owner: string,
	repo: string,
	newBranch: string,
	fromBranch: string,
): Promise<z.infer<typeof GitHubReferenceSchema>> {
	let sha: string;
	if (fromBranch) {
		sha = await getBranchSHA(secrets, owner, repo, fromBranch);
	} else {
		sha = await getDefaultBranchSHA(secrets, owner, repo);
	}

	return createBranch(secrets, owner, repo, {
		ref: newBranch,
		sha,
	});
}

export async function updateBranch(
	secrets: Record<string, string>,
	owner: string,
	repo: string,
	branch: string,
	sha: string,
): Promise<z.infer<typeof GitHubReferenceSchema>> {
	const response = await githubRequest(`https://api.github.com/repos/${owner}/${repo}/git/refs/heads/${branch}`, secrets, {
		method: 'PATCH',
		body: {
			sha,
			force: true,
		},
	});

	return GitHubReferenceSchema.parse(response);
}
