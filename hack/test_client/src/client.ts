// @ts-ignore
import WebSocket from "ws";
// @ts-ignore
global.WebSocket = WebSocket;

import { logger, newClient } from "@beamlit/sdk";
import { LocalToolkit } from "@beamlit/sdk/functions/local.js";
import { description, name, payload, url } from "./config.js";

const main = async () => {
  const client = newClient();
  const toolkit = new LocalToolkit(client, name, url);
  await toolkit.initialize(name, description);
  const functions = await toolkit.getTools();
  console.log(functions.map((f) => f.name));
  const tool = functions.find((f) => f.name === payload.name);
  if (!tool) {
    logger.error(`Tool with name ${payload.name} not found`);
    return;
  }
  const result = await tool.invoke(payload.arguments);
  logger.info(result);
};

main();
