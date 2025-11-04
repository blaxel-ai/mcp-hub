export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "get-config",
    arguments: {
    },
  }),
  () => ({
    name: `search-${process.env.AIRWEAVE_COLLECTION}`,
    arguments: {
      query: "What's in my collection ?",
    },
  }),
];

export const description = "Airweave description";
export const name = "airweave";
export const url = "http://localhost:1400";
