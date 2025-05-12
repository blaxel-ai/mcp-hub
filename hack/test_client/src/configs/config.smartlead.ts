export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "smartlead_fetch_all_clients",
    arguments: {
    },
  })
];

export const description = "Smartlead description";
export const name = "smartlead";
export const url = "http://localhost:8080";
