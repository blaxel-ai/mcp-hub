import { Tool } from "@modelcontextprotocol/sdk/types.js";

export interface UpdateIssueArgs {
  issueKey: string;
  title?: string;
  description?: string;
  assigneeEmail?: string;
  status?: string;
  priority?: number;
}

export const updateIssueTool: Tool = {
  name: "update_issue",
  description: `Updates an existing Linear issue with specified changes. Modify issue properties like title, description, assignee, status, or priority.

Example usage:
{
  "issueKey": "ENG-123",
  "title": "Updated: Implement OAuth authentication",
  "description": "## Updated Requirements\\n- Add Google OAuth\\n- Implement secure token storage",
  "assigneeEmail": "dev@company.com",
  "status": "In Progress",
  "priority": 1
}`,
  inputSchema: {
    type: "object",
    properties: {
      issueKey: {
        type: "string",
        description: "Issue Key to update (e.g., 'ENG-123')"
      },
      title: {
        type: "string",
        description: "New title for the issue"
      },
      description: {
        type: "string",
        description: "New description (supports markdown formatting)"
      },
      assigneeEmail: {
        type: "string",
        description: "Email of the user to assign the issue to"
      },
      status: {
        type: "string",
        description: "New status for the issue (e.g., 'In Progress', 'Done')"
      },
      priority: {
        type: "number",
        enum: [1, 2, 3, 4],
        description: "New priority level (1: Urgent, 2: High, 3: Normal, 4: Low)"
      }
    },
    required: ["issueKey"]
  }
};
