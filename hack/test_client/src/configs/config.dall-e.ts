export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [];

export const description = "Dall-E description";
export const name = "dall-e";
export const url = "http://localhost:8080";
