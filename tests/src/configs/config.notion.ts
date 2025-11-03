export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "API-get-users",
    arguments: {},
  })
];

export const description = "Notion description";
export const name = "notion";
export const url = "http://localhost:1400";
