import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { v4 as uuidv4 } from "uuid";
import { WebSocket, WebSocketServer } from "ws";

// Store the original connect method
const originalConnect = Server.prototype.connect;

// Override the connect method to use DebugTransport when environment variable is set
Server.prototype.connect = async function (transport) {
  // Use DebugTransport if DEBUG_TRANSPORT is set to 'true'
  console.error("Using WebSocketServerTransport instead of StdioServerTransport");
  const port = parseInt(process.env.BL_SERVER_PORT ?? '8080', 10);
  console.error(`Websocket server started on port ${port}`);
  const debugTransport = new WebSocketServerTransport(port);
  return originalConnect.call(this, debugTransport);
};

export class WebSocketServerTransport {
  constructor(port) {
    this.port = port;
    this.wss = new WebSocketServer({ port: this.port });
    this.clients = new Map();
  }

  set onmessage(handler) {
    this.messageHandler = handler ? (msg, clientId) => {
      if (msg.id === undefined) {
        return handler(msg);
      }
      return handler({
        ...msg,
        id: clientId + ":" + msg.id
      });
    } : undefined;
  }

  async start() {
    this.wss.on("connection", (ws) => {
      const clientId = uuidv4();
      this.clients.set(clientId, ws);
      this.onconnection?.(clientId);

      ws.on("message", (data) => {
        try {
          const msg = JSON.parse(data.toString());
          this.messageHandler?.(msg, clientId);
        } catch (err) {
          this.onerror?.(new Error(`Failed to parse message: ${err}`));
        }
      });

      ws.on("close", () => {
        this.clients.delete(clientId);
        this.ondisconnection?.(clientId);
      });

      ws.on("error", (err) => {
        this.onerror?.(err);
      });
    });
  }

  async send(msg) {
    const [cId, msgId] = msg.id?.split(":") ?? [];
    msg.id = parseInt(msgId);
    const data = JSON.stringify(msg);
    const deadClients = [];
    if (cId) {
      // Send to specific client
      const client = this.clients.get(cId);
      if (client?.readyState === WebSocket.OPEN) {
        client.send(data);
      } else {
        this.clients.delete(cId);
        this.ondisconnection?.(cId);
      }
    }

    for (const [id, client] of this.clients.entries()) {
      if (client.readyState !== WebSocket.OPEN) {
        deadClients.push(id);
      }
    }
    // Cleanup dead clients
    deadClients.forEach((id) => {
      this.clients.delete(id);
      this.ondisconnection?.(id);
    });
  }

  async broadcast(msg) {
    return this.send(msg);
  }

  async close() {
    return new Promise((resolve) => {
      this.wss.close(() => {
        this.clients.clear();
        resolve();
      });
    });
  }
}