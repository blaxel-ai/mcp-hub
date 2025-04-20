export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "brave_web_search",
    arguments: {
      query: "weather in San Francisco",
      count: 1,
      offset: 1
    },
  }),
];

export const description = "Brave search description";
export const name = "brave-search";
export const url = "http://localhost:8080";
