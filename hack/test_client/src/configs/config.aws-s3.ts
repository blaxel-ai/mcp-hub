export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "list_files",
    arguments: {
      bucket: "cli-blaxel-test",
    },
  }),
  () => {
    return {
      name: "upload_file",
      arguments: {
        bucket: "cli-blaxel-test",
        key: "test.txt",
        data: "Hello, world!",
      },
    };
  },
  () => {
    return {
      name: "retrieve_file",
      arguments: {
        bucket: "cli-blaxel-test",
        key: "test.txt",
      },
    };
  },
];

export const description = "AWS S3 description";
export const name = "aws";
export const url = "http://localhost:8080";
