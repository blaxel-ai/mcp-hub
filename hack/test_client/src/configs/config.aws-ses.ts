export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [];

export const description = "AWS SES description";
export const name = "aws-ses";
export const url = "http://localhost:8080";
