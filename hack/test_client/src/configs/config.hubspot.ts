export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "hubspot-get-user-details",
    arguments: {
    },
  })
];

export const description = "Hubspot description";
export const name = "hubspot";
export const url = "http://localhost:8080";
