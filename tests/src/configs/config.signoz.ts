export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "signoz_list_dashboards",
    arguments: {},
  }),
];

export const description = "SigNoz observability MCP server";
export const name = "signoz";
export const url = "http://localhost:1400";
