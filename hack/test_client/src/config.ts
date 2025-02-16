export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [];

export const description = "Qdrant description";
export const name = "qdrant";
export const url = "http://localhost:1400";
