export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "list_teams",
    arguments: {},
  }),
  () => ({
    name: "list_users",
    arguments: {},
  }),
  (res) => {
    const team = res.list_teams[0];
    return {
      name: "search_issues",
      arguments: {
        title: "test",
        teamKey: team.key,
      },
    };
  },
];

export const description = "Linear description";
export const name = "linear";
export const url = "http://localhost:8080";
