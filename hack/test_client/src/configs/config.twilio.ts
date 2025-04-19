export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "send_sms",
    arguments: {
      to: "+33610919161",
      message: "Hello, world!",
    },
  }),
];

export const description = "Qdrant description";
export const name = "qdrant";
export const url = "http://localhost:8080";
