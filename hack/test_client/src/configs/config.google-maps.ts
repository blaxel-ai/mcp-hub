export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "maps_search_places",
    arguments: {
      query: "el pirata",
      location: {},
      radius: 1
    },
  }),
];

export const description = "Google Maps description";
export const name = "google-maps";
export const url = "http://localhost:8080";
