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

export const description = "Notion description";
export const name = "notion";
export const url = "http://localhost:8080";
