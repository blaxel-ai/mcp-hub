import { Tool } from "@modelcontextprotocol/sdk/types.js";

export interface ListUsersArgs {
}

export const listUsersTool: Tool = {
	name: "list_users",
	description: `Lists all active users in the Linear organization. Use this to get user information and their identifiers.

Example usage:
{}`,
	inputSchema: {
		type: "object",
		properties: {
		}
	}
};
