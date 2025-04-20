export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [];

export const description = "Gmail description";
export const name = "gmail";
export const url = "http://localhost:8080";
