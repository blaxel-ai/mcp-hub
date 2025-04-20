export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "list_members",
    arguments: {
      server_id: "1347566164166971423",
    },
  }),
];

export const description = "Discord description";
export const name = "discord";
export const url = "http://localhost:8080";
