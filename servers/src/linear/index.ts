import { AddCommentArgs, addCommentTool } from './addComment.js';
import { LinearMCPClient } from './client.js';
import { CreateIssueArgs, createIssueTool } from './createIssue.js';
import { GetIssueDetailsArgs, getIssueDetailsTool } from './getIssueDetails.js';
import { GetUserIssuesArgs, getUserIssuesTool } from './getUserIssues.js';
import { ListTeamsArgs, listTeamsTool } from './listTeams.js';
import { ListUsersArgs, listUsersTool } from './listUsers.js';
import { SearchIssuesArgs, searchIssuesTool } from './searchIssues.js';
import { UpdateIssueArgs, updateIssueTool } from './updateIssue.js';

export async function call(request: Request, config: Record<string, string>, secrets: Record<string, string>) {
	try {
		const linearClient = new LinearMCPClient(config, secrets);
		const requestBody: { name: string; arguments: any } = await request.json() as { name: string; arguments: any };
		if (!requestBody.arguments) {
			throw new Error('No arguments provided');
		}
		switch (requestBody.name) {
			case 'search_issues': {
				const args = requestBody.arguments as SearchIssuesArgs;
				return await linearClient.searchIssues(args);
			}
			case 'get_issue_details': {
				const args = requestBody.arguments as GetIssueDetailsArgs;
				return await linearClient.getIssueDetails(args);
			}
			case 'create_issue': {
				const args = requestBody.arguments as CreateIssueArgs;
				return await linearClient.createIssue(args);
			}
			case 'list_teams': {
				const args = requestBody.arguments as ListTeamsArgs;
				return await linearClient.listTeams(args);
			}
			case 'list_users': {
				const args = requestBody.arguments as ListUsersArgs;
				return await linearClient.listUsers(args);
			}
			case 'update_issue': {
				const args = requestBody.arguments as UpdateIssueArgs;
				return await linearClient.updateIssue(args);
			}
			case 'add_comment': {
				const args = requestBody.arguments as AddCommentArgs;
				return await linearClient.addComment(args);
			}
			case 'get_user_issues': {
				const args = requestBody.arguments as GetUserIssuesArgs;
				return await linearClient.getUserIssues(args);
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
			isError: true,
		};
	}
}

export async function list() {
	return {
		tools: [
			searchIssuesTool,
			getIssueDetailsTool,
			createIssueTool,
			listTeamsTool,
			listUsersTool,
			addCommentTool,
			getUserIssuesTool,
			updateIssueTool,
		],
	};
}

export async function infos() {
	return {
		name: 'linear',
		displayName: 'Linear',
		integration: 'linear',
		categories: ['project-management'],
		description: 'Search, create and update issues in your teams',
		icon: 'data:image/png;base64,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',
		url: 'https://linear.app/settings/api/applications/new',
		form: {
			config: {},
			secrets: {
				apiKey: {
					description: 'Linear API key',
					label: 'Linear API key',
					required: true,
				},
			},
		},
		intructions: `This server provides access to Linear, a project management tool. Use it to manage issues, track work, and coordinate with teams.

Key capabilities:
- Create and update issues: Create new tickets or modify existing ones with titles, descriptions, priorities, and team assignments.
- Search functionality: Find issues across the organization using flexible search queries with team and user filters.
- Team coordination: Access team-specific issues and manage work distribution within teams.
- Issue tracking: Add comments and track progress through status updates and assignments.
- Organization overview: View team structures and user assignments across the organization.

Tool Usage:
- create_issue:
  - use teamId from linear-organization: resource
  - priority levels: 1=urgent, 2=high, 3=normal, 4=low
  - status must match exact Linear workflow state names (e.g., "In Progress", "Done")

- update_issue:
  - get issue IDs from search_issues or linear-issue:/// resources
  - only include fields you want to change
  - status changes must use valid state IDs from the team's workflow

- search_issues:
  - combine multiple filters for precise results
  - use labels array for multiple tag filtering
  - query searches both title and description
  - returns max 10 results by default

- get_user_issues:
  - omit userId to get authenticated user's issues
  - useful for workload analysis and sprint planning
  - returns most recently updated issues first

- add_comment:
  - supports full markdown formatting
  - use displayIconUrl for bot/integration avatars
  - createAsUser for custom comment attribution

Best practices:
- When creating issues:
  - Write clear, actionable titles that describe the task well (e.g., "Implement user authentication for mobile app")
  - Include concise but appropriately detailed descriptions in markdown format with context and acceptance criteria
  - Set appropriate priority based on the context (1=critical to 4=nice-to-have)
  - Always specify the correct team ID (default to the user's team if possible)

- When searching:
  - Use specific, targeted queries for better results (e.g., "auth mobile app" rather than just "auth")
  - Apply relevant filters when asked or when you can infer the appropriate filters to narrow results

- When adding comments:
  - Use markdown formatting to improve readability and structure
  - Keep content focused on the specific issue and relevant updates
  - Include action items or next steps when appropriate

- General best practices:
  - Fetch organization data first to get valid team IDs
  - Use search_issues to find issues for bulk operations
  - Include markdown formatting in descriptions and comments

Resource patterns:
- linear-issue:///{issueId} - Single issue details (e.g., linear-issue:///c2b318fb-95d2-4a81-9539-f3268f34af87)
- linear-team:///{teamId}/issues - Team's issue list (e.g., linear-team:///OPS/issues)
- linear-user:///{userId}/assigned - User assignments (e.g., linear-user:///USER-123/assigned)
- linear-organization: - Organization for the current user
- linear-viewer: - Current user context

The server uses the authenticated user's permissions for all operations.`,
	};
}
