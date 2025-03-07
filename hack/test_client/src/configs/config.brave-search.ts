export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [];

export const description = "Brave search description";
export const name = "brave-search";
export const url = "http://localhost:1400";
