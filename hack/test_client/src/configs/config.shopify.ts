export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "get-shop",
    arguments: {},
  }),
];

export const description = "Shopify description";
export const name = "shopify";
export const url = "http://localhost:8080";
