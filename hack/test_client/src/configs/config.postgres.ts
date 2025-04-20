export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [];

export const description = "Postgres description";
export const name = "postgres";
export const url = "http://localhost:8080";
