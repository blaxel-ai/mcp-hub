import { query } from "@anthropic-ai/claude-agent-sdk";

for await (const message of query({
  prompt: "List all available pets",
  options: {
    mcpServers: {
      petstore: {
        type: "http",
        url: "https://mcp-petstore-code-mode-uuqxla.runv2.blaxel.dev",
        headers: { "x-blaxel-authorization": `Bearer ${process.env.BL_API_KEY}` },
      },
    },
    allowedTools: ["mcp__petstore__*"],
  },
})) {
  if (message.type === "result" && message.subtype === "success") {
    console.log(message.result);
  }
}