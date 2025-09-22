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

export const description = "AWS S3 description";
export const name = "aws";
export const url = "http://localhost:1400";
