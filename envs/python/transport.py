import asyncio
import logging
import os
import socket
from contextlib import asynccontextmanager

import anyio
import mcp.server.sse as sse
import mcp.server.stdio as stdio
import mcp.server.websocket as websocket
import mcp.types as types
import uvicorn
from anyio.streams.memory import (MemoryObjectReceiveStream,
                                  MemoryObjectSendStream)
from starlette.applications import Starlette
from starlette.routing import WebSocketRoute
from starlette.types import Receive, Scope, Send
from starlette.websockets import WebSocket

logger = logging.getLogger(__name__)

@asynccontextmanager
async def websocket_server():
    # Create bidirectional streams for communication
    read_stream_writer, read_stream = anyio.create_memory_object_stream(0)
    write_stream, write_stream_reader = anyio.create_memory_object_stream(0)

    # Dictionnaire pour suivre les clients actifs
    active_clients = {}
    # Mutex pour protéger l'accès au dictionnaire
    clients_lock = asyncio.Lock()

    async def _websocket_server(websocket: WebSocket):
        # Generate a unique client ID
        client_id = str(id(websocket))

        # Create dedicated streams for this client
        client_read_stream_writer, client_read_stream = anyio.create_memory_object_stream(0)
        client_write_stream, client_write_stream_reader = anyio.create_memory_object_stream(0)

        # Register this client's streams
        async with clients_lock:
            active_clients[client_id] = (client_read_stream, client_write_stream)

        # Accept the connection without requiring a specific subprotocol
        await websocket.accept()

        # Use a flag to track if the websocket is already closed
        websocket_closed = False

        # Create a lock to prevent concurrent close operations
        close_lock = asyncio.Lock()

        async def safe_close_websocket():
            nonlocal websocket_closed
            async with close_lock:
                if not websocket_closed:
                    try:
                        await websocket.close()
                        websocket_closed = True
                    except Exception as e:
                        # Ignorer les erreurs spécifiques de fermeture déjà effectuée
                        if "already completed" not in str(e) and "once a close message has been sent" not in str(e):
                            logger.error(f"Error closing websocket: {e}")

        async def ws_reader():
            nonlocal websocket_closed
            try:
                async for message in websocket.iter_json():
                    try:
                        logger.debug(f"Client {client_id} received: {message}")
                        logger.debug("--------------------------------")
                        client_message = types.JSONRPCMessage.model_validate(message)

                        if hasattr(client_message.root, 'id'):
                            client_message.root.id = client_id + ":" + str(client_message.root.id)
                        # Transmettre le message au stream principal
                        await read_stream_writer.send(client_message)
                    except Exception as exc:
                        logger.error(f"Error processing message: {exc}")
                        await read_stream_writer.send(exc)
            except anyio.ClosedResourceError:
                logger.info(f"WebSocket reader stream closed for client {client_id}")
            except Exception as e:
                logger.error(f"WebSocket reader error for client {client_id}: {e}")
            finally:
                await safe_close_websocket()
                # Unregister this client when done
                async with clients_lock:
                    if client_id in active_clients:
                        del active_clients[client_id]

        async def ws_writer():
            nonlocal websocket_closed
            try:
                async for message in client_write_stream_reader:
                    logger.debug(f"Sending to client {client_id}: {message.root}")
                    logger.debug("--------------------------------")
                    if hasattr(message.root, 'id'):
                        if message.root.id.split(":")[0] != client_id:
                            continue
                        message.root.id = int(message.root.id.split(":")[1])
                    if websocket_closed:
                        break
                    try:
                        obj = message.model_dump(
                            by_alias=True, mode="json", exclude_none=True
                        )
                        logger.info(f"Sending message to client {client_id}: {obj}")
                        await websocket.send_json(obj)
                    except Exception as e:
                        if "once a close message has been sent" in str(e):
                            websocket_closed = True
                            break
                        logger.error(f"Error sending message to client {client_id}: {e}")
            except anyio.ClosedResourceError:
                logger.info(f"WebSocket writer stream closed for client {client_id}")
            except Exception as e:
                logger.error(f"WebSocket writer error for client {client_id}: {e}")
            finally:
                await safe_close_websocket()

        async with anyio.create_task_group() as tg:
            tg.start_soon(ws_reader)
            tg.start_soon(ws_writer)

    # Create Starlette app with WebSocket route
    routes = [
        WebSocketRoute("/", _websocket_server),
    ]
    app = Starlette(routes=routes, debug=True)

    # Configure the server with more detailed logging
    port = os.getenv("BL_SERVER_PORT", 8080)
    server = uvicorn.Server(
        config=uvicorn.Config(
            app=app,
            host="0.0.0.0",
            port=int(port),
            log_level="info",
        )
    )

    async def start_server():
        logger.info(f"Starting WebSocket server on 0.0.0.0:8080")
        await server.serve()

    # Create a multiplexer function to handle messages from all clients
    async def message_router():
        try:
            # Écouter les messages du stream principal
            async for message in write_stream_reader:
                logger.debug(f"Message from main stream: {message.root}")
                logger.debug("--------------------------------")

                # Envoyer le message à tous les clients actifs
                async with clients_lock:
                    client_items = list(active_clients.items())

                for client_id, (_, client_write_stream) in client_items:
                    try:
                        # Envoyer le message au client
                        await client_write_stream.send(message)
                    except Exception as e:
                        logger.error(f"Error sending message to client {client_id}: {e}")
        except anyio.ClosedResourceError:
            logger.info("Main write stream closed")
        except Exception as e:
            logger.error(f"Message router error: {e}")

    # Start the server in a background task
    async with anyio.create_task_group() as tg:
        tg.start_soon(start_server)
        tg.start_soon(message_router)  # Start the message router

        try:
            # Yield the original streams to maintain compatibility
            yield (read_stream, write_stream)
        finally:
            # Cancel all tasks when the context is exited
            tg.cancel_scope.cancel()
            logger.info("WebSocket server stopped")

# Assign the websocket_server to the different interfaces
stdio.stdio_server = websocket_server
websocket.websocket_server = websocket_server
sse.sse_server = websocket_server
