let namespaceId = "";
export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "get_kvs",
    arguments: {},
  }),
  // (res: Record<string, any>) => {
  //   namespaceId = res["get_kvs"][0].id;
  //   return {
  //     name: "kv_put",
  //     arguments: {
  //       namespaceId: namespaceId,
  //       key: "blaxel",
  //       value: "test",
  //     },
  //   };
  // },
  // (res: Record<string, any>) => {
  //   return {
  //     name: "kv_get",
  //     arguments: {
  //       namespaceId: namespaceId,
  //       key: "blaxel",
  //     },
  //   };
  // },
  // (res: Record<string, any>) => {
  //   return {
  //     name: "kv_delete",
  //     arguments: {
  //       namespaceId: namespaceId,
  //       key: "blaxel",
  //     },
  //   };
  // },
  (res: Record<string, any>) => {
    namespaceId = res["get_kvs"][0].id;
    return {
      name: "kv_list",
      arguments: {
        namespaceId: namespaceId,
      },
    };
  },
  (res: Record<string, any>) => {
    return {
      name: "worker_list",
      arguments: {},
    };
  },
  (res: Record<string, any>) => {
    return {
      name: "r2_list_buckets",
      arguments: {},
    };
  },
  (res: Record<string, any>) => {
    return {
      name: "d1_list_databases",
      arguments: {},
    };
  },
];

export const description = "Cloudflare description";
export const name = "cloudflare";
export const url = "http://localhost:8080";
