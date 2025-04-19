export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "list_contacts",
    arguments: {},
  }),
];

export const description = "Sendgrid description";
export const name = "sendgrid";
export const url = "http://localhost:8080";
