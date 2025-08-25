export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "search-web",
    arguments: {
      query: "How does the new EU AI Act affect startups?",
      depth: "standard",
    },
  }),
];

export const description = "Linkup Toolkit description";
export const name = "linkup";
export const url = "http://localhost:1400";
