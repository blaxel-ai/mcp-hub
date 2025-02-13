import { Tool } from "@modelcontextprotocol/sdk/types.js";

export interface CreateIssueArgs {
  title: string;
  teamKey: string;
  description?: string;
  assigneeEmail?: string;
  status?: string;
  priority?: number;
}

export const createIssueTool: Tool = {
  name: "create_issue",
  description: `Creates a new Linear issue with specified details. Use this to create tickets for tasks, bugs, or feature requests.

Example usage:
{
  "title": "Implement user authentication",
  "teamKey": "ENG",
  "description": "## Requirements\\n- Add OAuth support\\n- Implement login flow\\n- Add session management",
  "assigneeEmail": "developer@company.com",
  "status": "In Progress",
  "priority": 2
}`,
  inputSchema: {
    type: "object",
    properties: {
      title: { type: "string", description: "Clear and descriptive issue title" },
      teamKey: { type: "string", description: "Team key (e.g., 'ENG', 'DESIGN')" },
      description: { type: "string", description: "Detailed issue description (supports markdown)", optional: true },
      assigneeEmail: { type: "string", description: "Email of the team member to assign the issue to", optional: true },
      status: { type: "string", description: "Status of the issue (e.g., 'In Progress', 'Done')", optional: true },
      priority: { type: "number", description: "Priority level (1: Urgent, 2: High, 3: Normal, 4: Low)", optional: true }
    },
    required: ["title", "teamKey"]
  }
};
