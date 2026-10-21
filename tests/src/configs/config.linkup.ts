export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    // Renamed from "search-web" when upstream (LinkupPlatform) replaced the
    // archived Python server (python-mcp-server / mcp-search-linkup) with
    // linkup-mcp-server (TypeScript, npm). Same query/depth args shape.
    name: "linkup-search",
    arguments: {
      query: "How does the new EU AI Act affect startups?",
      depth: "standard",
    },
  }),
];

export const description = "Linkup Toolkit description";
export const name = "linkup";
export const url = "http://localhost:1400";
