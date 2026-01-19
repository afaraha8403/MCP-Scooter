# Contributing to MCP Scooter

First off, thank you for considering contributing to MCP Scooter! 🎉

MCP Scooter is built by [Balacode.io](https://balacode.io) and we welcome contributions from the community. This document will help you get started.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Submitting Changes](#submitting-changes)
- [Adding MCP Registry Entries](#adding-mcp-registry-entries)
- [Style Guidelines](#style-guidelines)

## Code of Conduct

This project adheres to our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior via GitHub issues.

## How Can I Contribute?

### 🐛 Reporting Bugs

Before creating a bug report, please check existing issues to avoid duplicates.

When filing a bug, include:
- A clear, descriptive title
- Steps to reproduce the issue
- Expected vs. actual behavior
- Your environment (OS, Go version, Node version)
- Relevant logs or screenshots

### 💡 Suggesting Features

Feature requests are welcome! Please:
- Check if the feature already exists or is planned
- Describe the problem you're trying to solve
- Explain your proposed solution
- Consider alternatives you've thought about

### 🔧 Pull Requests

We love pull requests! For major changes, please open an issue first to discuss what you'd like to change.

### 🎨 Adding MCP Definitions

One of the easiest ways to contribute is adding new MCP server definitions to the registry. See [Adding MCP Registry Entries](#adding-mcp-registry-entries) below.

## Development Setup

### Prerequisites

| Tool | Version | Installation |
|------|---------|--------------|
| Go | 1.24+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 18+ | [nodejs.org](https://nodejs.org/) |
| Rust | Latest | [rustup.rs](https://rustup.rs/) |
| Make | Any | Usually pre-installed |

### Getting Started

```bash
# Clone the repository
git clone https://github.com/balacode/mcp-scout.git
cd mcp-scout

# Install dependencies
make deps

# Run validation to ensure everything works
make validate

# Run tests
make test

# Start development mode
make dev
```

### Building

```bash
# Build Go binaries
make build

# Full CI check (format, lint, validate, test, build)
make ci
```

## Project Structure

```
MCP Scooter/
├── appdata/
│   ├── clients/        # AI client configurations
│   ├── registry/       # MCP server definitions (organized by source)
│   │   └── official/   # Official MCP definitions (JSON)
│   └── schemas/        # JSON Schema for validation
├── cmd/
│   ├── scooter/        # Main application entry point
│   └── validate-registry/  # Registry validation CLI
├── desktop/            # Tauri + React frontend
│   ├── src/            # React components
│   └── src-tauri/      # Rust/Tauri backend
├── internal/
│   ├── api/            # HTTP/SSE server
│   └── domain/         # Core business logic
│       ├── discovery/  # Tool discovery engine
│       ├── integration/# Client integrations
│       ├── profile/    # Profile management
│       └── registry/   # Registry validation
└── .doc/               # Documentation & specs
```

## Submitting Changes

### Commit Messages

We follow conventional commits:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code change
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:
```
feat(registry): add linear-mcp definition
fix(gateway): handle SSE connection timeout
docs(readme): update installation instructions
```

### Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Make your changes
4. Run `make ci` to ensure all checks pass
5. Commit your changes with a descriptive message
6. Push to your fork
7. Open a Pull Request

### PR Checklist

- [ ] Code follows the project's style guidelines
- [ ] Tests pass (`make test`)
- [ ] Validation passes (`make validate`)
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventional commits

## Adding MCP Registry Entries

This is one of the most valuable contributions! Here's how:

### 1. Create the JSON file

Create `appdata/registry/official/{name}.json`:

```json
{
  "$schema": "../../schemas/mcp-registry.schema.json",
  "name": "your-tool-name",
  "version": "1.0.0",
  "title": "Your Tool Name",
  "description": "A brief description of what this tool does",
  "category": "development",
  "source": "official",
  "authorization": {
    "type": "api_key",
    "required": true,
    "env_var": "YOUR_API_KEY",
    "display_name": "API Key",
    "help_url": "https://example.com/get-api-key"
  },
  "tools": [
    {
      "name": "tool_action",
      "title": "Tool Action",
      "description": "What this tool action does",
      "inputSchema": {
        "type": "object",
        "properties": {
          "param": {
            "type": "string",
            "description": "Parameter description"
          }
        },
        "required": ["param"]
      }
    }
  ],
  "package": {
    "type": "npm",
    "name": "@scope/package-name"
  },
  "about": "# Your Tool Name\n\nMarkdown documentation..."
}
```

### 2. Add an icon (optional but recommended)

- Place SVG icon in `desktop/public/registry-logos/{name}.svg`
- Reference it in JSON as `"icon": "/registry-logos/{name}.svg"`

### 3. Validate

```bash
make validate
```

### 4. Submit PR

See the [full registry specification](.doc/mcp-registry-specification.md) for all available fields.

## Style Guidelines

### Go

- Follow standard Go conventions (`go fmt`, `go vet`)
- Use meaningful variable names
- Add comments for exported functions
- Write tests for new functionality

### TypeScript/React

- Functional components with hooks
- TypeScript strict mode
- Descriptive component and prop names

### JSON

- 2-space indentation
- Include `$schema` reference
- Include `about` field with documentation

---

## Questions?

Feel free to open an issue or start a discussion. We're here to help!

Thank you for contributing to MCP Scooter! 🚀
