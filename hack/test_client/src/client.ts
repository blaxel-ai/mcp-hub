import { getFunctions, logger } from "@beamlit/sdk";
import { LocalToolkit } from "@beamlit/sdk/functions/local.js";
import { Client as ModelContextProtocolClient } from "@modelcontextprotocol/sdk/client/index.js";
import { WebSocketClientTransport } from "@beamlit/sdk/functions/transport/websocket.js";
import { MCPClient, MCPToolkit } from "@beamlit/sdk";
import { createClient } from "@hey-api/client-fetch";
import yargs from "yargs";
import { hideBin } from 'yargs/helpers'


const main = async () => {
    const argv = yargs(hideBin(process.argv))
    .option('name', {
      type: 'string',
      description: 'Name of the function'
    })
    .option('description', {
      type: 'string',
      description: 'Description of the function'
    })
    .option('url', {
      type: 'string',
      description: 'URL of the function'
    })
    .option('list-tools', {
      type: 'boolean',
      description: 'List all tools'
    })
    .option('call-tool', {
      type: 'string',
      description: 'Call a tool with the given name'
    })
    .option('call-tool-args', {
      type: 'object',
      description: 'Call a tool with the given name and arguments'
    })
    .help()
    .parseSync()
  
  if (argv.listTools) {
    const functions = await getFunctions({ localFunctions: [{ name: argv.name, description: argv.description, url: argv.url }] });
    functions.forEach((f) => {
      logger.info(f.name);
    logger.info(f.description);
    logger.debug(f);
  });

  } else {}
//     const modelContextProtocolClient = new ModelContextProtocolClient(
//         {
//             name: argv.name,
//             version: "1.0.0",
//         },
//         {
//             capabilities: {
//                 tools: {}
//             }
//         }
//     );

//     const transport = new WebSocketClientTransport(new URL(argv.url), {
//         "x-beamlit-authorization": "",
//         "x-beamlit-workspace": "",
//     });
//     await modelContextProtocolClient.connect(transport);
//     const mcpClient = new MCPClient(modelContextProtocolClient);
//     const client = createClient({
//         baseUrl: argv.url,
//         headers: {
//             "x-beamlit-authorization": "",
//             "x-beamlit-workspace": "",
//         },
//     });
//     const toolkit = new LocalToolkit(client, argv.name, argv.url);
//     const mcpToolkit = new MCPToolkit(mcpClient);
//     await mcpToolkit.initialize();
//     const functions = await toolkit.getTools();
//     const tool = functions.find((f) => f.name === argv.callTool);
//     if (!tool) {
//       logger.error(`Tool with name ${argv.callTool} not found`);
//       return;
//     }
//     const result = await tool.invoke(argv.callToolArgs);
//     logger.info(result);
//   }
};


main();
