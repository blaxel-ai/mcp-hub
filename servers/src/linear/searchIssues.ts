import { Tool } from "@modelcontextprotocol/sdk/types.js";

export interface SearchIssuesArgs {
  query: string;
  teamKey?: string;
  status?: string;
  assigneeEmail?: string;
  assigneeName?: string;
  labels?: string[];
  priority?: number;
  includeArchived?: boolean;
  estimate?: number;
  limit?: number;
}

export const searchIssuesTool: Tool = {
	name: "search_issues",
	description: `Search Linear issues with powerful filtering options. Find issues by text, team, status, assignee, labels, and priority.

Example usage:
{
  "query": "authentication bug",
  "teamKey": "ENG",
  "status": "In Progress",
  "assigneeEmail": "dev@company.com",
  "labels": ["bug", "critical"],
  "priority": 1,
  "limit": 10
}`,
	inputSchema: {
	  type: "object",
	  properties: {
      query: {
        type: "string",
        description: "Search text to match in issue titles and descriptions"
      },
      teamKey: {
        type: "string",
        description: "Team key to filter issues by (e.g., 'ENG')",
        optional: true
      },
      status: {
        type: "string",
        description: "Filter by issue status (e.g., 'In Progress', 'Done')",
        optional: true
      },
      assigneeEmail: {
        type: "string",
        description: "User email to filter issues by assignee",
        optional: true
      },
      assigneeName: {
        type: "string",
        description: "User name to filter issues by assignee",
        optional: true
      },
      labels: {
        type: "array",
        items: { type: "string" },
        description: "List of label names to filter issues by",
        optional: true
      },
      priority: {
        type: "number",
        enum: [1, 2, 3, 4],
        description: "Filter by priority level (1: Urgent, 2: High, 3: Normal, 4: Low)",
        optional: true
      },
      includeArchived: {
        type: "boolean",
        description: "Include archived issues in search results (default: false)",
        optional: true
      },
      estimate: {
        type: "number",
        description: "Filter by estimate (in points)",
        optional: true
      },
      limit: {
        type: "number",
        description: "Maximum number of issues to return (default: 10)",
        optional: true
      }
	  },
	  required: ["query"]
	}
};
