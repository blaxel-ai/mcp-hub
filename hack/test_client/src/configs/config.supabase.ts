export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "list_projects",
    arguments: {
    },
  }),
];

export const description = "Example MCP for Supabase";
export const name = "supabase";
export const url = "http://localhost:8080";
