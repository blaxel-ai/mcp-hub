import { Tool } from "@modelcontextprotocol/sdk/types.js";

export interface ListTeamsArgs {
}

export const listTeamsTool: Tool = {
  name: "list_teams",
  description: `Lists all teams in the Linear organization. Use this to get team information and their identifiers.

Example usage:
{
}`,
  inputSchema: {
    type: "object",
    properties: {
    },
    required: []
  }
};
