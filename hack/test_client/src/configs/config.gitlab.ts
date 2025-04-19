export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [];

export const description = "Gitlab description";
export const name = "gitlab";
export const url = "http://localhost:8080";
