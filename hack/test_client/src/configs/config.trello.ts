export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "get_lists",
    arguments: {
    },
  })
];

export const description = "Trello description";
export const name = "trello";
export const url = "http://localhost:8080";
