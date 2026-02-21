# MCP Server Template - TypeScript

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Node.js 18+](https://img.shields.io/badge/node-18+-green.svg)](https://nodejs.org/downloads/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.8+-blue.svg)](https://www.typescriptlang.org/)
[![MCP SDK](https://img.shields.io/badge/MCP_SDK-1.20+-brightgreen.svg)](https://github.com/modelcontextprotocol/typescript-sdk)

</div>

A template implementation of a Model Context Protocol (MCP) server in TypeScript using the official [MCP TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk).
This template demonstrates how to build MCP-compatible servers with HTTP transport that can be integrated with MCP-enabled clients for enhanced AI interactions and tool integrations.

## 📑 Table of Contents

- [MCP Server Template - TypeScript](#mcp-server-template---typescript)
  - [📑 Table of Contents](#-table-of-contents)
  - [✨ Features](#-features)
  - [🚀 Quick Start](#-quick-start)
  - [📋 Prerequisites](#-prerequisites)
  - [💻 Installation](#-installation)
  - [🔧 Usage](#-usage)
    - [Running the Server Locally](#running-the-server-locally)
    - [Testing your server](#testing-your-server)
  - [📁 Project Structure](#-project-structure)
  - [❓ Troubleshooting](#-troubleshooting)
    - [Common Issues](#common-issues)
  - [👥 Contributing](#-contributing)
  - [🆘 Support](#-support)
  - [📄 License](#-license)

## ✨ Features

- Official MCP TypeScript SDK implementation
- HTTP transport using Streamable HTTP (latest MCP protocol)
- Session management for multiple concurrent connections
- Tool registration with Zod schema validation
- Built with TypeScript 5.8+ for modern type safety
- Express.js server for production-ready HTTP handling
- Easy debugging with MCP Inspector
- Compatible with all MCP-enabled clients

## 🚀 Quick Start

For those who want to get up and running quickly:

```bash
# Clone the repository
git clone https://github.com/yourusername/template-mcp-ts.git

# Navigate to the project directory
cd template-mcp-ts

# Install dependencies
npm install

# Start the MCP server
npm run dev

# In another terminal, test with MCP Inspector
npm run inspect
```

## 📋 Prerequisites

- **Node.js:** 18.0 or later
- **npm, yarn, or pnpm:** Package manager of your choice

## 💻 Installation

**Clone the repository and install dependencies:**

```bash
git clone https://github.com/yourusername/template-mcp-ts.git
cd template-mcp-ts
npm install
```

**Environment Variables (optional):**

- `PORT` - Server port (default: 1338)
- `HOST` - Server host (default: 0.0.0.0)

## 🔧 Usage

### Running the Server Locally

Start the development server with hot reloading:

```bash
npm run dev
```

For production build and run:

```bash
# Build TypeScript to JavaScript
npm run build

# Run the compiled server
npm run prod
```

_Note:_ The development server automatically restarts when you make changes to the source code.

### Testing your server

You can test your MCP server using the MCP Inspector:

```bash
npm run inspect
```

Or manually:

```bash
npx @modelcontextprotocol/inspector --transport http --server-url http://localhost:1338/mcp
```

This launches the MCP Inspector UI which helps debug server functionality through a web interface.
You can connect to your MCP server and test your Tools/Resources/Prompts.

The server exposes an HTTP endpoint at `/mcp` that accepts MCP protocol requests.

## 📁 Project Structure

- **src/server.ts** - Main MCP server implementation with HTTP transport
- **package.json** - npm package configuration
- **tsconfig.json** - TypeScript compiler configuration
- **dist/** - Compiled JavaScript output (generated after build)

## ❓ Troubleshooting

### Common Issues

1. **Port Already in Use**:
   - Change the port by setting the `PORT` environment variable
   - Example: `PORT=3001 npm run dev`
   - Check what's using port 1338: `lsof -i :1338` (Mac/Linux)

2. **MCP Protocol Errors**:
   - Ensure you're using MCP SDK version 1.20.0 or later
   - Check that the MCP Inspector version is compatible
   - Review server startup logs for initialization errors

3. **Tool Registration Problems**:
   - Ensure tool schemas are properly defined with Zod
   - Check tool argument validation
   - Verify tool handlers return the correct response format

4. **Connection Issues**:
   - Verify the server is running: `curl http://localhost:1338/mcp`
   - Check for CORS issues if connecting from a browser
   - Ensure the `/mcp` endpoint path is correct

5. **MCP Inspector Connection**:
   - Verify the server URL includes `/mcp` path
   - Check that the transport is set to `http`
   - Ensure no firewall is blocking the connection

6. **TypeScript Compilation Issues**:
   - Run `npm run build` to check for compilation errors
   - Ensure all dependencies are installed: `npm install`
   - Verify Node.js version is 18 or later

For more help, please [submit an issue](https://github.com/yourusername/template-mcp-ts/issues) on GitHub.

## 👥 Contributing

Contributions are welcome! Here's how you can contribute:

1. **Fork** the repository
2. **Create** a feature branch:
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **Commit** your changes:
   ```bash
   git commit -m 'Add amazing feature'
   ```
4. **Push** to the branch:
   ```bash
   git push origin feature/amazing-feature
   ```
5. **Submit** a Pull Request

Please make sure to update tests as appropriate and follow the MCP protocol standards.

## 🆘 Support

If you need help with this template:

- [Submit an issue](https://github.com/blaxel-templates/template-mcp-ts/issues) for bug reports or feature requests
- Visit the [Blaxel Documentation](https://docs.blaxel.ai) for platform guidance
- Check the [Model Context Protocol Documentation](https://modelcontextprotocol.io/) for protocol specifications
- Join our [Discord Community](https://discord.gg/G3NqzUPcHP) for real-time assistance

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
