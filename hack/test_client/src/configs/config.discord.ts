export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  // () => ({
  //   name: "list_files",
  //   arguments: {
  //     bucket: "cli-blaxel-test",
  //   },
  // }),
];

export const description = "Discord description";
export const name = "discord";
export const url = "http://localhost:1400";
