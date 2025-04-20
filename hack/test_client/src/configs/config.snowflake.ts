export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "execute_query",
    arguments: {
      query: "SELECT 1",
    },
  }),
];

export const description = "Snowflake description";
export const name = "snowflake";
export const url = "http://localhost:8080";
