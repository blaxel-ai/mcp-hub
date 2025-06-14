export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "query",
    arguments: {
      sql: "SELECT * FROM \"mcp-test-table\"",
    },
  }),
];

export const description = "Postgres description";
export const name = "postgres";
export const url = "http://localhost:1400";
