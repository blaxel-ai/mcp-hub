export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "list_documentation",
    arguments: {
    },
  }),

];

export const description = "Nia Toolkit description";
export const name = "nia";
export const url = "http://localhost:1400";
