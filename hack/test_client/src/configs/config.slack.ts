export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "slack_list_channels",
    arguments: {
      limit: 10,
    },
  }),
];

export const description = "Github description";
export const name = "github";
export const url = "http://localhost:8080";
