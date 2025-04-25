export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "tavily-search",
    arguments: {
      query: "What is the weather in Paris?",
    },
  }),
];

export const description = "Tavily description";
export const name = "tavily";
export const url = "http://localhost:8080";
