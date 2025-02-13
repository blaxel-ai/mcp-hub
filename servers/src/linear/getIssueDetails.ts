import { Tool } from "@modelcontextprotocol/sdk/types.js";

export interface GetIssueDetailsArgs {
  issueKey: string;
}

export const getIssueDetailsTool: Tool = {
	name: "get_issue_details",
	description: `Get comprehensive details of a Linear issue by its Key. Use this to retrieve full information about specific tickets.

Example usage:
{
  "issueKey": "ENG-123"
}`,
	inputSchema: {
	  type: "object",
	  properties: {
      issueKey: {
        type: "string",
        description: "Linear issue Key (e.g., 'ENG-123')"
      }
	  },
	  required: ["issueKey"]
	}
};
