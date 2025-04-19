export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "search_repositories",
    arguments: {
      query: "beamlit"
    },
  }),
];

export const description = "Github description";
export const name = "github";
export const url = "http://localhost:8080";
