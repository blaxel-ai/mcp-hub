export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "resolve-library-id",
    arguments: {
      libraryName: "blaxel"
    },
  }),
  () => ({
    name: "get-library-docs",
    arguments: {
      context7CompatibleLibraryID: "/blaxel-ai/docs",
      topic: 'agent'
    },
  }),
];

export const description = "Context7 Toolkit description";
export const name = "context7";
export const url = "http://localhost:1400";
