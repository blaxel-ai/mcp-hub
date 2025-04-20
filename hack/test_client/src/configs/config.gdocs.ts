export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [];

export const description = "Google Docs description";
export const name = "gdocs";
export const url = "http://localhost:8080";
