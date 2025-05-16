export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "postgrestRequest",
    arguments: {
      method: "GET",
      path: "/mcp-test-table"
    },
  }),
];

export const description = "Example MCP for Supabase Postgrest";
export const name = "supabase-postgrest";
export const url = "http://localhost:8080";
