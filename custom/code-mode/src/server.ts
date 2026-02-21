import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import "@blaxel/telemetry";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { SandboxInstance } from "@blaxel/core";
import express from "express";
import { z } from "zod";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { parse as parseYaml } from "yaml";

/** Checks whether a string looks like an HTTP(S) URL. */
function isUrl(value: string): boolean {
  return value.startsWith("http://") || value.startsWith("https://");
}

/** Checks whether a string is likely base64-encoded content (alphanumeric+/= and long enough). */
function isBase64(value: string): boolean {
  return /^[A-Za-z0-9+/=]+$/.test(value) && value.length > 64;
}

/** Attempts to decode a base64 string as JSON; returns the raw string on failure. */
function parseReference(raw: string): any {
  try {
    return JSON.parse(Buffer.from(raw, "base64").toString("utf-8"));
  } catch {
    return raw;
  }
}

type ReferenceResult = { spec: any; source: string };

/** Converts an OpenAPI security scheme name to its corresponding environment variable name. */
function schemeToEnvVar(name: string): string {
  return `AUTH_${name.replace(/[^a-zA-Z0-9]/g, "_").toUpperCase()}`;
}

/**
 * Extracts authentication environment variables based on the security schemes
 * defined in the OpenAPI spec. For each scheme, looks for an env var named
 * AUTH_<SCHEME_NAME>. Returns the found values and the full list of expected env var names.
 */
function extractAuthEnvs(spec: any): { envs: Record<string, string>; expected: string[] } {
  const schemes =
    spec?.components?.securitySchemes || // OpenAPI 3.x
    spec?.securityDefinitions ||         // OpenAPI 2.x
    {};

  const envs: Record<string, string> = {};
  const expected: string[] = [];

  for (const [name] of Object.entries<any>(schemes)) {
    const envVar = schemeToEnvVar(name);
    expected.push(envVar);
    const value = process.env[envVar];
    if (value) {
      envs[envVar] = value;
    }
  }

  return { envs, expected };
}

/**
 * Builds a human-readable description of the active authentication schemes,
 * including concrete usage examples for each type (bearer, basic, apiKey, oauth2).
 * Only includes schemes whose env var is actually set.
 */
function describeAuthSchemes(spec: any, activeEnvs: Record<string, string>): string {
  const schemes =
    spec?.components?.securitySchemes ||
    spec?.securityDefinitions ||
    {};

  const entries = Object.entries<any>(schemes).filter(
    ([name]) => schemeToEnvVar(name) in activeEnvs
  );
  if (entries.length === 0) return "";

  const lines = entries.map(([name, scheme]) => {
    const envVar = schemeToEnvVar(name);
    if (scheme.type === "http" && scheme.scheme === "bearer") {
      return `- ${envVar}: Bearer token. Use as: headers: { "Authorization": \`Bearer \${process.env.${envVar}}\` }`;
    }
    if (scheme.type === "http" && scheme.scheme === "basic") {
      return `- ${envVar}: Basic auth credentials (base64-encoded user:pass). Use as: headers: { "Authorization": \`Basic \${process.env.${envVar}}\` }`;
    }
    if (scheme.type === "apiKey" && scheme.in === "header") {
      return `- ${envVar}: API key. Use as: headers: { "${scheme.name}": process.env.${envVar} }`;
    }
    if (scheme.type === "apiKey" && scheme.in === "query") {
      return `- ${envVar}: API key. Append to URL as: ?${scheme.name}=\${process.env.${envVar}}`;
    }
    if (scheme.type === "oauth2" || scheme.type === "openIdConnect") {
      return `- ${envVar}: OAuth token. Use as: headers: { "Authorization": \`Bearer \${process.env.${envVar}}\` }`;
    }
    return `- ${envVar}: Auth credential for scheme "${name}" (type: ${scheme.type || "unknown"})`;
  });

  return `\nAuthentication (available via process.env in the sandbox):\n${lines.join("\n")}`;
}

/**
 * Resolves a reference string into a parsed spec object.
 * Supports three formats:
 * - URL (http/https): fetches and parses as JSON or YAML
 * - Base64: decodes and parses as JSON
 * - File path: reads from disk and parses as JSON or YAML based on extension
 */
async function resolveReference(ref: string, label: string): Promise<ReferenceResult> {
  if (isUrl(ref)) {
    const res = await fetch(ref);
    if (!res.ok) throw new Error(`Failed to fetch reference from ${ref}: ${res.status}`);
    const text = await res.text();
    let spec: any;
    try {
      spec = JSON.parse(text);
    } catch {
      spec = parseYaml(text);
    }
    return { spec, source: `URL: ${ref} (${label})` };
  }

  if (isBase64(ref)) {
    return { spec: parseReference(ref), source: `base64 (${label})` };
  }

  const filePath = resolve(ref);
  const content = readFileSync(filePath, "utf-8");
  const spec = ref.endsWith(".json") ? JSON.parse(content) : parseYaml(content);
  return { spec, source: `file: ${filePath} (${label})` };
}

const HTTP_METHODS = ["get", "post", "put", "patch", "delete"] as const;

/** Collects all unique tags from every operation in the spec, sorted alphabetically. */
function extractTags(spec: any): string[] {
  const tags = new Set<string>();
  for (const methods of Object.values<any>(spec?.paths || {})) {
    for (const method of HTTP_METHODS) {
      for (const tag of methods?.[method]?.tags || []) {
        tags.add(tag);
      }
    }
  }
  return [...tags].sort();
}

interface SampleEndpoints {
  listEndpoint: { path: string; op: any } | null;
  getEndpoint: { path: string; op: any } | null;
  postEndpoint: { path: string; op: any } | null;
  taggedEndpoint: { path: string; method: string; tag: string } | null;
}

/**
 * Picks representative endpoints from the spec for use in tool description examples.
 * Selects a list endpoint (GET on a collection), a generic GET, a POST with body, and a tagged endpoint.
 */
function pickSampleEndpoints(spec: any): SampleEndpoints {
  const paths = Object.entries<any>(spec?.paths || {});
  let listEndpoint: SampleEndpoints["listEndpoint"] = null;
  let getEndpoint: SampleEndpoints["getEndpoint"] = null;
  let postEndpoint: SampleEndpoints["postEndpoint"] = null;
  let taggedEndpoint: SampleEndpoints["taggedEndpoint"] = null;

  for (const [path, methods] of paths) {
    if (!listEndpoint && methods.get && !path.endsWith("}")) {
      listEndpoint = { path, op: methods.get };
    }
    if (!getEndpoint && methods.get) {
      getEndpoint = { path, op: methods.get };
    }
    if (!postEndpoint && methods.post?.requestBody) {
      postEndpoint = { path, op: methods.post };
    }
    if (!taggedEndpoint) {
      for (const method of HTTP_METHODS) {
        const op = methods?.[method];
        if (op?.tags?.[0]) {
          taggedEndpoint = { path, method, tag: op.tags[0] };
          break;
        }
      }
    }
    if (listEndpoint && getEndpoint && postEndpoint && taggedEndpoint) break;
  }

  return { listEndpoint, getEndpoint, postEndpoint, taggedEndpoint };
}

/** Returns a single auth header snippet (e.g. `"Authorization": \`Bearer ...\``) for use in code examples. */
function buildExampleAuthHeader(spec: any, authEnvs: Record<string, string>): string {
  const schemes = spec?.components?.securitySchemes || spec?.securityDefinitions || {};
  for (const [name, scheme] of Object.entries<any>(schemes)) {
    const envVar = schemeToEnvVar(name);
    if (!(envVar in authEnvs)) continue;

    if (scheme.type === "http" && scheme.scheme === "bearer") {
      return `"Authorization": \`Bearer \${process.env.${envVar}}\``;
    }
    if (scheme.type === "http" && scheme.scheme === "basic") {
      return `"Authorization": \`Basic \${process.env.${envVar}}\``;
    }
    if (scheme.type === "apiKey" && scheme.in === "header") {
      return `"${scheme.name}": process.env.${envVar}`;
    }
    if (scheme.type === "oauth2" || scheme.type === "openIdConnect") {
      return `"Authorization": \`Bearer \${process.env.${envVar}}\``;
    }
  }
  return "";
}

/**
 * Builds the dynamic description for the search tool based on the loaded spec.
 * Includes API name, available tags, TypeScript interfaces, and concrete code examples
 * using real paths extracted from the spec.
 */
function buildSearchDescription(spec: any): { description: string; codeDescription: string } {
  const apiName = spec?.info?.title || "the API";
  const tags = extractTags(spec);
  const { getEndpoint, postEndpoint, taggedEndpoint } = pickSampleEndpoints(spec);
  const pathCount = Object.keys(spec?.paths || {}).length;

  const parts: string[] = [];
  parts.push(`Search the ${apiName} OpenAPI spec. All $refs are pre-resolved inline.`);

  if (tags.length > 0) {
    const shown = tags.slice(0, 30);
    const suffix = tags.length > 30 ? ` ... (${tags.length} total)` : "";
    parts.push(`\nTags: ${shown.join(", ")}${suffix}`);
  }

  parts.push(`\nPaths: ${pathCount} total`);

  parts.push(`

Types:

interface OperationInfo {
  summary?: string;
  description?: string;
  tags?: string[];
  parameters?: Array<{ name: string; in: string; required?: boolean; schema?: unknown; description?: string }>;
  requestBody?: { required?: boolean; content?: Record<string, { schema?: unknown }> };
  responses?: Record<string, { description?: string; content?: Record<string, { schema?: unknown }> }>;
}

interface PathItem {
  get?: OperationInfo;
  post?: OperationInfo;
  put?: OperationInfo;
  patch?: OperationInfo;
  delete?: OperationInfo;
}

declare const spec: {
  paths: Record<string, PathItem>;
};`);

  parts.push(`\n\nExamples:`);

  if (taggedEndpoint) {
    parts.push(`

// Find endpoints by tag
async () => {
  const results = [];
  for (const [path, methods] of Object.entries(spec.paths)) {
    for (const [method, op] of Object.entries(methods)) {
      if (op.tags?.some(t => t.toLowerCase() === '${taggedEndpoint.tag.toLowerCase()}')) {
        results.push({ method: method.toUpperCase(), path, summary: op.summary });
      }
    }
  }
  return results;
}`);
  }

  const detailEndpoint = getEndpoint;
  if (detailEndpoint) {
    parts.push(`

// Get endpoint details
async () => {
  const op = spec.paths['${detailEndpoint.path}']?.get;
  return { summary: op?.summary, parameters: op?.parameters };
}`);
  }

  if (postEndpoint) {
    parts.push(`

// Get endpoint with requestBody schema (refs are resolved)
async () => {
  const op = spec.paths['${postEndpoint.path}']?.post;
  return { summary: op?.summary, requestBody: op?.requestBody };
}`);
  }

  const codeDescription =
    "JavaScript async arrow function. Has access to `spec` (OpenAPI object with .paths, .components, .servers, .info). " +
    "Return the data you need — e.g. `() => Object.keys(spec.paths)` or " +
    (detailEndpoint
      ? `\`() => spec.paths['${detailEndpoint.path}'].get\`.`
      : "`() => spec.paths['/some/path'].get`.");

  return { description: parts.join(""), codeDescription };
}

/**
 * Builds the dynamic description for the execute tool based on the loaded spec.
 * Includes API name, server URLs, authentication instructions, and a concrete
 * fetch() example using a real endpoint and the correct auth header format.
 */
function buildExecuteDescription(
  spec: any,
  serverDescription: string,
  authEnvs: Record<string, string>,
  authDescription: string,
): string {
  const apiName = spec?.info?.title || "the API";
  const baseUrl = spec?.servers?.[0]?.url || "";
  const { listEndpoint } = pickSampleEndpoints(spec);
  const authHeader = buildExampleAuthHeader(spec, authEnvs);

  const parts: string[] = [];

  parts.push(
    `Execute JavaScript code against ${apiName}. ` +
    `First use the 'search' tool to find the right endpoints, then write code using fetch().`
  );
  parts.push(serverDescription);
  parts.push(authDescription);
  parts.push(`\n\nYour code must be an async arrow function that returns the result.`);

  if (listEndpoint && baseUrl) {
    const headersContent = authHeader ? `\n      ${authHeader}\n    ` : "";
    parts.push(`

Example:
async () => {
  const res = await fetch(\`${baseUrl}${listEndpoint.path}\`, {
    headers: {${headersContent}}
  });
  return res.json();
}`);
  }

  return parts.join("");
}

/**
 * Loads the API reference spec from the OPENAPI_REFERENCE env var (URL, base64, or file path),
 * defaulting to reference.yaml in the current directory.
 */
async function loadReference(): Promise<ReferenceResult> {
  const ref = process.env.OPENAPI_REFERENCE || "reference.yaml";
  return resolveReference(ref, "env");
}

/**
 * Bootstraps the Code Mode MCP server:
 * 1. Loads and parses the API reference
 * 2. Extracts and validates authentication env vars
 * 3. Registers search and execute tools
 * 4. Starts the Express HTTP server
 */
async function main() {
  const { spec, source } = await loadReference();
  const endpointCount = spec?.paths ? Object.values(spec.paths).reduce(
    (sum: number, methods: any) =>
      sum + Object.keys(methods).filter((m) =>
        ["get", "post", "put", "patch", "delete", "head", "options"].includes(m)
      ).length,
    0
  ) : 0;
  const { envs: authEnvs, expected: expectedAuthEnvs } = extractAuthEnvs(spec);
  const authDescription = describeAuthSchemes(spec, authEnvs);
  const servers = spec?.servers || [];
  let serverDescription = "";
  if (servers.length === 1) {
    serverDescription = `\nAPI base URL: ${servers[0].url}${servers[0].description ? ` (${servers[0].description})` : ""}`;
  } else if (servers.length > 1) {
    const lines = servers.map((s: any, i: number) => {
      const label = i === 0 ? " (default)" : " (fallback or situational)";
      return `  - ${s.url}${s.description ? ` (${s.description})` : ""}${label}`;
    });
    serverDescription = `\nAPI servers (use the first one by default, others may serve as fallbacks or for specific use cases):\n${lines.join("\n")}`;
  }

  const searchDesc = buildSearchDescription(spec);
  const executeDesc = buildExecuteDescription(spec, serverDescription, authEnvs, authDescription);

  console.log(`Loaded reference from ${source}`);
  if (spec?.info?.title) {
    console.log(`  API: ${spec.info.title}${spec.info.version ? ` v${spec.info.version}` : ""}`);
  }
  if (spec?.servers?.length) {
    for (const s of spec.servers) {
      console.log(`  Server: ${s.url}${s.description ? ` (${s.description})` : ""}`);
    }
  }
  console.log(`  Endpoints: ${endpointCount}`);
  const authKeys = Object.keys(authEnvs);
  if (authKeys.length > 0) {
    console.log(`  Authentication: ${authKeys.join(", ")}`);
  }
  if (expectedAuthEnvs.length > 0 && authKeys.length === 0) {
    throw new Error(
      `No authentication environment variable set. The API reference defines the following auth schemes: ${expectedAuthEnvs.join(", ")}. ` +
      `Set at least one of them before starting the server.`
    );
  }

  const server = new McpServer({
    name: "CodeModeServer",
    version: "1.0.0",
  });

  server.registerTool(
    "search",
    {
      description: searchDesc.description,
      inputSchema: {
        code: z.string().describe(searchDesc.codeDescription),
      },
    },
    async ({ code }) => {
      console.log(`[search] called: ${code.length > 120 ? code.slice(0, 120) + "…" : code}`);
      try {
        const fn = new Function(
          "spec",
          `return (async () => { const __fn = ${code}; return await __fn(); })()`
        );
        const result = await fn(spec);
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          isError: true,
        };
      }
    }
  );

  server.registerTool(
    "execute",
    {
      description: executeDesc,
      inputSchema: {
        code: z
          .string()
          .describe(
            "JavaScript async arrow function to execute in the sandbox. Has access to Node.js built-ins including fetch() for HTTP requests."
          ),
        timeout: z
          .number()
          .describe(
            "Timeout in seconds for the execution. Defaults to 30 seconds."
          )
          .optional(),
      },
    },
    async ({ code, timeout = 30 }: { code: string; timeout?: number }) => {
      console.log(`[execute] called (timeout=${timeout}s): ${code.length > 1200000 ? code.slice(0, 1200000) + "…" : code}`);
      try {
        const rawName = `code-mode-${process.env.BL_NAME || "local"}`;
        const sandboxName = rawName.length > 48 ? rawName.slice(0, 48) : rawName;
        const sandbox = await SandboxInstance.createIfNotExists({
          name: sandboxName,
          labels: {
            "code-mode": "true",
            "code-mode-name": process.env.BL_NAME ,
          },
          lifecycle: {
            expirationPolicies: [
              {
                type: "ttl-idle",
                action: "delete",
                value: "10m",
              },
            ],
          },
        });

        const script = [
          `const fn = ${code};`,
          `(async () => {`,
          `  try {`,
          `    const result = await fn();`,
          `    if (result !== undefined) process.stdout.write(JSON.stringify(result, null, 2));`,
          `  } catch (e) {`,
          `    process.stderr.write(e.stack || String(e));`,
          `    process.exit(1);`,
          `  }`,
          `})();`,
        ].join("\n");

        const encoded = Buffer.from(script).toString("base64");
        const result = await sandbox.process.exec({
          env: authEnvs,
          command: `node -e "eval(Buffer.from('${encoded}','base64').toString())"`,
          waitForCompletion: true,
          timeout,
        });

        if (result.exitCode !== 0) {
          return {
            content: [
              { type: "text", text: `Error: ${result.stderr || result.logs}` },
            ],
            isError: true,
          };
        }

        return {
          content: [
            { type: "text", text: result.stdout || "Executed successfully" },
          ],
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Sandbox error: ${err.message}` }],
          isError: true,
        };
      }
    }
  );

  const app = express();
  app.use(express.json());

  app.post("/mcp", async (req, res) => {
    const transport = new StreamableHTTPServerTransport({
      sessionIdGenerator: undefined,
      enableJsonResponse: true,
    });
    res.on("close", () => transport.close());
    await server.connect(transport);
    await transport.handleRequest(req, res, req.body);
  });

  const port = parseInt(process.env.PORT || "80");
  const host = process.env.HOST || "0.0.0.0";

  app.listen(port, () => {
    console.log(`Code Mode MCP Server running on http://${host}:${port}/mcp`);
  }).on("error", (error) => {
    console.error("Server error:", error);
    process.exit(1);
  });
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
