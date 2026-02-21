# Code Mode – Deploy any API as an MCP server on Blaxel

Code Mode turns any OpenAPI spec into a two-tool MCP server: **search** lets an AI agent explore the API spec, and **execute** runs JavaScript code in a sandbox to call the actual API. No backend code to write — just point it at a spec and deploy.

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                          AI Agent                                    │
│                                                                      │
│   "List all pets"  ──►  search() ──►  execute()                      │
└──────────┬───────────────────┬──────────────┬────────────────────────┘
           │                   │              │
           │  MCP (HTTP)       │              │
           ▼                   ▼              ▼
┌──────────────────────────────────────────────────────────────────────┐
│                     Code Mode MCP Server                             │
│                     (Blaxel MCP)                                      │
│                                                                      │
│  ┌─────────────────────┐    ┌──────────────────────────────────────┐ │
│  │   search tool        │    │   execute tool                       │ │
│  │                      │    │                                      │ │
│  │  Evaluates JS code   │    │  1. Creates/reuses sandbox           │ │
│  │  against in-memory   │    │  2. Injects auth env vars            │ │
│  │  OpenAPI spec        │    │  3. Runs JS code (fetch calls)       │ │
│  │                      │    │  4. Returns stdout / stderr          │ │
│  │  No network calls    │    │                                      │ │
│  │  Instant results     │    │                                      │ │
│  └─────────────────────┘    └──────────────┬───────────────────────┘ │
│                                             │                        │
└─────────────────────────────────────────────┼────────────────────────┘
                                              │
                                              ▼
                               ┌──────────────────────────┐
                               │   Blaxel Sandbox          │
                               │   (Node.js runtime)       │
                               │                           │
                               │   - Auto-scaling           │
                               │   - 10min idle TTL         │
                               │   - Auth env vars injected │
                               └──────────────┬────────────┘
                                              │
                                              │  fetch()
                                              ▼
                               ┌──────────────────────────┐
                               │   Target API              │
                               │   (e.g. Petstore,         │
                               │    Stripe, GitHub...)      │
                               └──────────────────────────┘
```

### Flow

1. The **agent** receives a user prompt (e.g. "List all pets")
2. It calls **search** to discover relevant endpoints in the OpenAPI spec
3. It calls **execute** with a `fetch()` script targeting the right endpoint
4. Code Mode spins up a **sandbox**, injects auth credentials, runs the code
5. The result flows back to the agent, which formats the answer

## What you get

- **search** tool: the agent can browse paths, tags, schemas, and request/response bodies from the OpenAPI spec
- **execute** tool: the agent writes and runs `fetch()` calls in a sandboxed Node.js environment with auth pre-configured

## Prerequisites

- [Blaxel CLI installed](https://docs.blaxel.ai/Get-started#)
- [Logged in](https://docs.blaxel.ai/Get-started#)

## Quick start with the Petstore API

### 1. Create the manifest

Create a file called `manifest.yaml`:

```yaml
apiVersion: blaxel.ai/v1alpha1
kind: MCP
metadata:
  displayName: Petstore Code Mode
  name: petstore-code-mode
spec:
  runtime:
    type: mcp
    image: blaxel/code-mode:latest
    generation: mk3
    envs:
      - name: OPENAPI_REFERENCE
        value: https://petstore3.swagger.io/api/v3/openapi.json
```

### 2. Deploy

```bash
bl apply -f manifest.yaml
```

### 3. Verify

```bash
bl get mcp petstore-code-mode
bl logs mcps petstore-code-mode
```

Once status shows `DEPLOYED`, any agent in your workspace can use `petstore-code-mode` as a tool provider.

## Adding authentication

Code Mode auto-detects security schemes defined in the OpenAPI spec and maps them to environment variables named `AUTH_<SCHEME_NAME>`.

For example, if the spec defines a scheme called `ApiKeyAuth`:

```yaml
    envs:
      - name: OPENAPI_REFERENCE
        value: https://api.example.com/openapi.json
      - name: AUTH_APIKEYAUTH
        value: ${secrets.AUTH_APIKEYAUTH}
```

### Supported auth types

| OpenAPI scheme type       | Env var usage                                      |
|---------------------------|----------------------------------------------------|
| `http` / `bearer`         | `Authorization: Bearer <token>`                    |
| `http` / `basic`          | `Authorization: Basic <base64>`                    |
| `apiKey` (header)         | Custom header with the key value                   |
| `apiKey` (query)          | Appended as query parameter                        |
| `oauth2` / `openIdConnect`| `Authorization: Bearer <token>`                    |

## Using it with the Claude Agent SDK

Once deployed, you can connect to your Code Mode MCP directly from TypeScript using the Claude Agent SDK:

```typescript
import { query } from "@anthropic-ai/claude-agent-sdk";

for await (const message of query({
  prompt: "List all available pets",
  options: {
    mcpServers: {
      petstore: {
        type: "http",
        url: "<your-mcp-url>/mcp",
        headers: { "x-blaxel-authorization": "Bearer <your-api-key>" },
      },
    },
    allowedTools: ["mcp__petstore__*"],
  },
})) {
  if (message.type === "result" && message.subtype === "success") {
    console.log(message.result);
  }
}
```

Find your MCP url:
- Console: [https://app.blaxel.ai](https://app.blaxel.ai)
- CLI: `bl get mcp petstore-code-mode -o json | jq -r '.[] | .metadata.url'`

Configure an Api Key:
- https://app.blaxel.ai/profile/security

## How it works under the hood

When an agent calls **search**, Code Mode evaluates a JavaScript function against the in-memory OpenAPI spec — no network calls, instant results. All `$ref` pointers are pre-resolved so the agent sees fully expanded schemas.

When an agent calls **execute**, the code runs inside an auto-scaling sandbox with a 10-minute idle TTL. Auth env vars are injected automatically. The agent typically does a search first to find the right endpoint, then executes a fetch call.

## Useful commands

```bash
bl get mcps                              # list all MCPs
bl get mcp petstore-code-mode            # check deployment status
bl delete mcp petstore-code-mode         # remove it
bl logs mcp petstore-code-mode           # view logs
```
