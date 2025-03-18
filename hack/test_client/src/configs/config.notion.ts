export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "notion_retrieve_bot_user",
    arguments: {},
  }),
  () => ({
    name: "notion_list_all_users",
    arguments: {},
  }),
  () => ({
    name: "notion_search",
    arguments: {
      query: "Product Documentation",
    },
  }),
];

export const description = "Discord description";
export const name = "discord";
export const url = "http://localhost:1400";
