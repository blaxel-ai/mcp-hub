export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "search",
    arguments: {
      query: "What is the capital of France?",
    },
  }),
];

export const description = "Blaxel Search description";
export const name = "blaxel-search";
export const url = "http://localhost:8080";
