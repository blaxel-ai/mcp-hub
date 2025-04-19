// @ts-ignore
import WebSocket from "ws";
// @ts-ignore
global.WebSocket = WebSocket;

import { logger, newClient } from "@blaxel/sdk";
import { LocalToolkit } from "@blaxel/sdk/functions/local";
import { exit } from "process";
import { name, payload, url } from "./configs/config.exa"; // TODO: Update here like: /configs/config.exa

const main = async () => {
  const client = newClient();
  const toolkit = new LocalToolkit(client, name, url);
  await toolkit.initialize(name);
  const functions = await toolkit.getTools();
  for (const fn of functions) {
    let params: string[] = [];
    if (fn.schema && "shape" in fn.schema) {
      const schema = fn.schema.shape;
      params = Object.keys(schema);
    }
    logger.info(`${fn.name} (${params.join(", ")})`);
  }

  let previousResult: Record<string, any> = {};
  for (const fn of payload) {
    const params = fn(previousResult);
    const tool = functions.find((f) => f.name === params.name);
    if (!tool) {
      logger.error(`Tool with name ${params.name} not found`);
      exit(1);
    }
    try {
      const result = await tool.invoke(params.arguments);

      const parsedResult = JSON.parse(result);
      if (parsedResult.length > 0 && parsedResult[0].text) {
        try {
          previousResult[params.name] = JSON.parse(parsedResult[0].text);
        } catch {
          previousResult[params.name] = parsedResult[0].text;
        }
        console.log(`Result: ${params.name}`, previousResult[params.name]);
      } else {
        console.log(`Result: ${params.name}`, parsedResult);
      }
    } catch (error) {
      console.log(`Error: ${params.name}`, error);
    }
  }
  exit(0);
};

main();
