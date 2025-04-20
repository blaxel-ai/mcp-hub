export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "qdrant_store_memory",
    arguments: {
      query: "My name is Christophe",
    },
  }),
  (res: Record<string, any>) => ({
    name: "qdrant_find_memories",
    arguments: {
      query: "My name",
    },
  }),
];

export const description = "Qdrant description";
export const name = "qdrant";
export const url = "http://localhost:8080";
