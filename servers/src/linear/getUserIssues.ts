import { Tool } from "@modelcontextprotocol/sdk/types.js";

export interface GetUserIssuesArgs {
  userEmail?: string;
  includeArchived?: boolean;
  limit?: number;
}

export const getUserIssuesTool: Tool = {
  name: "get_user_issues",
  description: `Retrieves issues assigned to a specific user or the authenticated user if no userEmail is provided. Returns issues sorted by last updated.

Example usage:
{
  "userEmail": "developer@company.com",
  "includeArchived": false,
  "limit": 50
}`,
  inputSchema: {
    type: "object",
    properties: {
      userEmail: {
        type: "string",
        description: "User email to fetch issues"
      },
      includeArchived: {
        type: "boolean",
        description: "Include archived issues in results (default: false)",
        optional: true
      },
      limit: {
        type: "number",
        description: "Maximum number of issues to return (default: 50)",
        optional: true
      }
    },
    required: ["userEmail"]
  }
};

