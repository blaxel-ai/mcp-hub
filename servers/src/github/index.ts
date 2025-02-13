#!/usr/bin/env node
import { z } from 'zod';
import { zodToJsonSchema } from 'zod-to-json-schema';

import {
	GitHubAuthenticationError,
	GitHubConflictError,
	GitHubError,
	GitHubPermissionError,
	GitHubRateLimitError,
	GitHubResourceNotFoundError,
	GitHubValidationError,
	isGitHubError,
} from './common/errors';
import * as branches from './operations/branches';
import * as commits from './operations/commits';
import * as files from './operations/files';
import * as issues from './operations/issues';
import * as pulls from './operations/pulls';
import * as repository from './operations/repository';
import * as search from './operations/search';

function formatGitHubError(error: GitHubError): string {
	let message = `GitHub API Error: ${error.message}`;

	if (error instanceof GitHubValidationError) {
		message = `Validation Error: ${error.message}`;
		if (error.response) {
			message += `\nDetails: ${JSON.stringify(error.response)}`;
		}
	} else if (error instanceof GitHubResourceNotFoundError) {
		message = `Not Found: ${error.message}`;
	} else if (error instanceof GitHubAuthenticationError) {
		message = `Authentication Failed: ${error.message}`;
	} else if (error instanceof GitHubPermissionError) {
		message = `Permission Denied: ${error.message}`;
	} else if (error instanceof GitHubRateLimitError) {
		message = `Rate Limit Exceeded: ${error.message}\nResets at: ${error.resetAt.toISOString()}`;
	} else if (error instanceof GitHubConflictError) {
		message = `Conflict: ${error.message}`;
	}

	return message;
}

export async function infos() {
	return {
		name: 'github',
		displayName: 'GitHub',
		categories: ['code', 'git'],
		integration: 'github',
		description: 'Search repos, files and issues, and commit in GitHub',
		icon: 'https://github.githubassets.com/favicons/favicon.svg',
		url: 'https://github.com/settings/personal-access-tokens',
		form: {
			config: {},
			secrets: {
				personalAccessToken: {
					description: 'Personal Access Token',
					label: 'Personal Access Token',
					required: true,
				},
			},
		},
	};
}

export async function list() {
	return {
		tools: [
			{
				name: 'create_branch',
				description: 'Create a new branch in a GitHub repository',
				inputSchema: zodToJsonSchema(branches.CreateBranchSchema),
			},
			{
				name: 'create_branch_from_ref',
				description: 'Create a new branch in a GitHub repository from a reference branch',
				inputSchema: zodToJsonSchema(branches.CreateBranchFromRefSchema),
			},
			{
				name: 'update_branch',
				description: 'Update a branch in a GitHub repository',
				inputSchema: zodToJsonSchema(branches.UpdateBranchSchema),
			},
			{
				name: 'get_branch_sha',
				description: 'Get the SHA of a branch in a GitHub repository',
				inputSchema: zodToJsonSchema(branches.GetBranchSHA),
			},
			{
				name: 'list_commits',
				description: 'Get list of commits of a branch in a GitHub repository',
				inputSchema: zodToJsonSchema(commits.ListCommitsSchema),
			},
			{
				name: 'create_or_update_file',
				description: 'Create or update a single file in a GitHub repository',
				inputSchema: zodToJsonSchema(files.CreateOrUpdateFileSchema),
			},
			{
				name: 'get_file_contents',
				description: 'Get the contents of a file or directory from a GitHub repository',
				inputSchema: zodToJsonSchema(files.GetFileContentsSchema),
			},
			{
				name: 'push_files',
				description: 'Push multiple files to a GitHub repository in a single commit',
				inputSchema: zodToJsonSchema(files.PushFilesSchema),
			},
			{
				name: 'create_issue',
				description: 'Create a new issue in a GitHub repository',
				inputSchema: zodToJsonSchema(issues.CreateIssueSchema),
			},
			{
				name: 'list_issues',
				description: 'List issues in a GitHub repository with filtering options',
				inputSchema: zodToJsonSchema(issues.ListIssuesOptionsSchema),
			},
			{
				name: 'update_issue',
				description: 'Update an existing issue in a GitHub repository',
				inputSchema: zodToJsonSchema(issues.UpdateIssueOptionsSchema),
			},
			{
				name: 'add_issue_comment',
				description: 'Add a comment to an existing issue',
				inputSchema: zodToJsonSchema(issues.IssueCommentSchema),
			},
			{
				name: 'get_issue',
				description: 'Get details of a specific issue in a GitHub repository.',
				inputSchema: zodToJsonSchema(issues.GetIssueSchema),
			},
			{
				name: 'search_repositories',
				description: 'Search for GitHub repositories',
				inputSchema: zodToJsonSchema(repository.SearchRepositoriesSchema),
			},
			{
				name: 'create_repository',
				description: 'Create a new GitHub repository in your account',
				inputSchema: zodToJsonSchema(repository.CreateRepositoryOptionsSchema),
			},
			{
				name: 'fork_repository',
				description: 'Fork a GitHub repository to your account or specified organization',
				inputSchema: zodToJsonSchema(repository.ForkRepositorySchema),
			},
			{
				name: 'create_pull_request',
				description: 'Create a new pull request in a GitHub repository',
				inputSchema: zodToJsonSchema(pulls.CreatePullRequestSchema),
			},
			{
				name: 'get_pull_request',
				description: 'Get details of a specific pull request in a GitHub repository',
				inputSchema: zodToJsonSchema(pulls.GetPullRequestReviewsSchema),
			},
			{
				name: 'list_pull_requests',
				description: 'List pull requests in a GitHub repository',
				inputSchema: zodToJsonSchema(pulls.ListPullRequestsSchema),
			},
			{
				name: 'create_pull_request_review',
				description: 'Create a review for a pull request',
				inputSchema: zodToJsonSchema(pulls.CreatePullRequestReviewSchema),
			},
			{
				name: 'merge_pull_request',
				description: 'Merge a pull request',
				inputSchema: zodToJsonSchema(pulls.MergePullRequestSchema),
			},
			{
				name: 'get_pull_request_files',
				description: 'Get the files changed in a pull request',
				inputSchema: zodToJsonSchema(pulls.GetPullRequestFilesSchema),
			},
			{
				name: 'get_pull_request_comments',
				description: 'Get comments on a pull request',
				inputSchema: zodToJsonSchema(pulls.GetPullRequestCommentsSchema),
			},
			{
				name: 'get_pull_request_reviews',
				description: 'Get reviews for a pull request',
				inputSchema: zodToJsonSchema(pulls.GetPullRequestReviewsSchema),
			},
			{
				name: 'get_pull_request_status',
				description: 'Get the status of a pull request',
				inputSchema: zodToJsonSchema(pulls.GetPullRequestStatusSchema),
			},
			{
				name: 'search_code',
				description: 'Search for code across GitHub repositories',
				inputSchema: zodToJsonSchema(search.SearchCodeSchema),
			},
			{
				name: 'search_issues',
				description: 'Search for issues and pull requests across GitHub repositories',
				inputSchema: zodToJsonSchema(search.SearchIssuesSchema),
			},
			{
				name: 'search_users',
				description: 'Search for users on GitHub',
				inputSchema: zodToJsonSchema(search.SearchUsersSchema),
			},
		],
	};
}

export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	try {
		const requestBody: { name: string; arguments: any } = await request.json() as { name: string; arguments: any };
		if (!requestBody.arguments) {
			throw new Error('Arguments are required');
		}

		let result: any;
		switch (requestBody.name) {
			// Branches operations
			case 'create_branch': {
				const args = branches.CreateBranchSchema.parse(requestBody.arguments);
				result = await branches.createBranch(secrets, args.owner, args.repo, {
					ref: args.branch,
					sha: args.options?.sha,
				});
				break;
			}
			case 'create_branch_from_ref': {
				const args = branches.CreateBranchFromRefSchema.parse(requestBody.arguments);
				result = await branches.createBranchFromRef(secrets, args.owner, args.repo, args.branch, args.from_branch);
				break;
			}
			case 'update_branch': {
				const args = branches.UpdateBranchSchema.parse(requestBody.arguments);
				result = await branches.updateBranch(secrets, args.owner, args.repo, args.branch, args.sha);
				break;
			}
			case 'get_branch_sha': {
				const args = branches.GetBranchSHA.parse(requestBody.arguments);
				result = await branches.getDefaultBranchSHA(secrets, args.owner, args.repo);
				break;
			}

			// Commits operations
			case 'list_commits': {
				const args = commits.ListCommitsSchema.parse(requestBody.arguments);
				result = await commits.listCommits(secrets, args.owner, args.repo, args.page, args.perPage, args.sha);
				break;
			}

			// Files operations
			case 'get_file_contents': {
				const args = files.GetFileContentsSchema.parse(requestBody.arguments);
				result = await files.getFileContents(secrets, args.owner, args.repo, args.path, args.branch);
				break;
			}
			case 'create_or_update_file': {
				const args = files.CreateOrUpdateFileSchema.parse(requestBody.arguments);
				result = await files.createOrUpdateFile(
					secrets,
					args.owner,
					args.repo,
					args.path,
					args.content,
					args.message,
					args.branch,
					args.sha,
				);
				break;
			}
			case 'push_files': {
				const args = files.PushFilesSchema.parse(requestBody.arguments);
				result = await files.pushFiles(secrets, args.owner, args.repo, args.branch, args.files, args.message);
				break;
			}

			// Issues operations
			case 'create_issue': {
				const args = issues.CreateIssueSchema.parse(requestBody.arguments);
				const { owner, repo, ...options } = args;
				result = await issues.createIssue(secrets, owner, repo, options);
				break;
			}
			case 'get_issue': {
				const args = issues.GetIssueSchema.parse(requestBody.arguments);
				result = await issues.getIssue(secrets, args.owner, args.repo, args.issue_number);
				break;
			}
			case 'add_issue_comment': {
				const args = issues.IssueCommentSchema.parse(requestBody.arguments);
				const { owner, repo, issue_number, body } = args;
				result = await issues.addIssueComment(secrets, owner, repo, issue_number, body);
				break;
			}
			case 'list_issues': {
				const args = issues.ListIssuesOptionsSchema.parse(requestBody.arguments);
				const { owner, repo, ...options } = args;
				result = await issues.listIssues(secrets, owner, repo, options);
				break;
			}
			case 'update_issue': {
				const args = issues.UpdateIssueOptionsSchema.parse(requestBody.arguments);
				const { owner, repo, issue_number, ...options } = args;
				result = await issues.updateIssue(secrets, owner, repo, issue_number, options);
				break;
			}
			// Pull operations
			case 'create_pull_request': {
				const args = pulls.CreatePullRequestSchema.parse(requestBody.arguments);
				result = await pulls.createPullRequest(secrets, args);
				break;
			}
			case 'get_pull_request': {
				const args = pulls.GetPullRequestReviewsSchema.parse(requestBody.arguments);
				result = await pulls.getPullRequest(secrets, args.owner, args.repo, args.pull_number);
				break;
			}
			case 'list_pull_requests': {
				const args = pulls.ListPullRequestsSchema.parse(requestBody.arguments);
				result = await pulls.listPullRequests(secrets, args.owner, args.repo, { ...args });
				break;
			}
			case 'create_pull_request_review': {
				const args = pulls.CreatePullRequestReviewSchema.parse(requestBody.arguments);
				result = await pulls.createPullRequestReview(secrets, args.owner, args.repo, args.pull_number, args);
				break;
			}
			case 'merge_pull_request': {
				const args = pulls.MergePullRequestSchema.parse(requestBody.arguments);
				result = await pulls.mergePullRequest(secrets, args.owner, args.repo, args.pull_number, args);
				break;
			}
			case 'get_pull_request_files': {
				const args = pulls.GetPullRequestFilesSchema.parse(requestBody.arguments);
				result = await pulls.getPullRequestFiles(secrets, args.owner, args.repo, args.pull_number);
				break;
			}
			case 'get_pull_request_comments': {
				const args = pulls.GetPullRequestCommentsSchema.parse(requestBody.arguments);
				result = await pulls.getPullRequestComments(secrets, args.owner, args.repo, args.pull_number);
				break;
			}
			case 'get_pull_request_reviews': {
				const args = pulls.GetPullRequestReviewsSchema.parse(requestBody.arguments);
				result = await pulls.getPullRequestReviews(secrets, args.owner, args.repo, args.pull_number);
				break;
			}
			case 'get_pull_request_status': {
				const args = pulls.GetPullRequestStatusSchema.parse(requestBody.arguments);
				result = await pulls.getPullRequestStatus(secrets, args.owner, args.repo, args.pull_number);
				break;
			}

			// Repository operations
			case 'search_repositories': {
				const args = repository.SearchRepositoriesSchema.parse(requestBody.arguments);
				result = await repository.searchRepositories(secrets, args.query, args.page, args.perPage);
				break;
			}
			case 'create_repository': {
				const args = repository.CreateRepositoryOptionsSchema.parse(requestBody.arguments);
				result = await repository.createRepository(secrets, args);
				break;
			}
			case 'fork_repository': {
				const args = repository.ForkRepositorySchema.parse(requestBody.arguments);
				result = await repository.forkRepository(secrets, args.owner, args.repo, args.organization);
				break;
			}

			// Search operations
			case 'search_code': {
				const args = search.SearchCodeSchema.parse(requestBody.arguments);
				result = await search.searchCode(secrets, args);
				break;
			}

			case 'search_issues': {
				const args = search.SearchIssuesSchema.parse(requestBody.arguments);
				result = await search.searchIssues(secrets, args);
				break;
			}
			case 'search_users': {
				const args = search.SearchUsersSchema.parse(requestBody.arguments);
				result = await search.searchUsers(secrets, args);
				break;
			}

			default:
				throw new Error(`Unknown tool: ${requestBody.name}`);
		}
		return {
			isError: false,
			content: [{ type: 'text', text: JSON.stringify(result, null, 2) }],
		};
	} catch (error) {
		if (error instanceof z.ZodError) {
			return {
				isError: true,
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
		if (isGitHubError(error)) {
			return {
				isError: true,
				content: [
					{
						type: 'text',
						text: JSON.stringify({ error: formatGitHubError(error) }, null, 2),
					},
				],
			};
		}
		throw error;
	}
}
