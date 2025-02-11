# MCP hub Importer

A command-line interface (CLI) tool for importing Model Context Protocol servers (MCPs) from configuration files.

## Features

- Import MCPs from YAML configuration files
- Selectively import specific MCPs
- Push container images to registries
- Validate MCP configurations

## Installation

```bash
go install github.com/beamlit/mcp-hub@latest
```

## Usage

### Import all MCPs from config

```bash
mcp-hub import --config hub.yaml
```

### Import a specific MCP

```bash
mcp-hub import --config hub.yaml --mcp <mcp-name>
```

### Push images to registry

```bash
mcp-hub import --config hub.yaml --push
```

## Configuration

Create a `hub.yaml` file to define your MCPs. Example configuration:

```yaml
repositories:
  github-smithery-reference-servers:
    repository: https://github.com/smithery-ai/reference-servers.git
    smitheryPath: src/github/smithery.yaml
    dockerfile: src/github/Dockerfile
    packageManager: npm
    branch: main
    displayName: GitHub
    icon: https://github.com/smithery-ai/reference-servers/blob/main/src/github/gtasks-mcp/logo.jpg
    description: A collection of reference servers for GitHub.
    longDescription: A collection of reference servers for GitHub.
    tags:
      - reference-servers
      - smithery
    categories:
      - gtasks-mcp
      - gtasks-mcp-smithery
      - gtasks-mcp-smithery-reference-servers
  
  brave-search:
    repository: https://github.com/smithery-ai/reference-servers.git
    smitheryPath: src/brave-search/smithery.yaml
    dockerfile: src/brave-search/Dockerfile
    branch: main
    displayName: Brave Search
    icon: https://github.com/brave/brave-search/blob/main/src/brave/logo.jpg
    description: A search engine for Brave.
    longDescription: A search engine for Brave.
    tags:
      - brave-search
      - brave-search-smithery
      - brave-search-smithery-reference-servers
    categories:
      - brave-search
      - brave-search-smithery
      - brave-search-smithery-reference-servers
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the [Apache License 2.0](LICENSE).

## Acknowledgements

- [smithery](https://smithery.ai/) - For establishing the MCP package standard
- All our contributors and supporters

## Support

If you encounter any issues or have questions, please file an issue on our [GitHub repository](https://github.com/beamlit/mcp-hub/issues).
