export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "web_search",
    arguments: {
      query: "What is the capital of France?",
    },
  }),
];

export const description = "Exa description";
export const name = "exa";
export const url = "http://localhost:8080";
