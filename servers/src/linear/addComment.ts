import { Tool } from "@modelcontextprotocol/sdk/types.js";

export interface AddCommentArgs {
  issueKey: string;
  comment: string;
  userEmail?: string;
}

export const addCommentTool: Tool = {
  name: "add_comment",
  description: `Adds a comment to an existing Linear issue. Use this to add comments to tickets for tasks, bugs, or feature requests.

Example usage:
{
  "issueKey": "CHA-52",
  "comment": "This is a test comment",
  "userEmail": "user@example.com"
}
`,
  inputSchema: {
    type: "object",
    properties: {
      issueKey: { type: "string", description: "Issue Key (e.g., 'CHA-52')" },
      comment: { type: "string", description: "Comment text (supports markdown formatting)" },
      userEmail: { type: "string", description: "Optional User Email to attribute the comment to", optional: true }
    },
    required: ["issueKey", "comment"]
  }
};
