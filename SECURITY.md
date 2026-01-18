# Security Policy

## Supported Versions

MCP Scooter is currently in active development. Security updates will be applied to the latest version.

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest | :x:               |

## Reporting a Vulnerability

We take security seriously at MCP Scooter. If you discover a security vulnerability, please report it responsibly.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of these methods:

1. **GitHub Security Advisories** (Preferred)
   - Go to the repository's Security tab
   - Click "Report a vulnerability"
   - Fill out the form with details

2. **Direct Contact**
   - Open a private security advisory on GitHub
   - Include as much detail as possible

### What to Include

When reporting a vulnerability, please include:

- **Description** of the vulnerability
- **Steps to reproduce** the issue
- **Potential impact** of the vulnerability
- **Suggested fix** (if you have one)
- **Your contact information** for follow-up questions

### What to Expect

- **Acknowledgment:** We will acknowledge receipt within 48 hours
- **Initial Assessment:** We will provide an initial assessment within 7 days
- **Resolution Timeline:** We aim to resolve critical issues within 30 days
- **Credit:** We will credit you in the security advisory (unless you prefer to remain anonymous)

### Safe Harbor

We consider security research conducted in accordance with this policy to be:

- Authorized concerning any applicable anti-hacking laws
- Authorized concerning any relevant anti-circumvention laws
- Exempt from restrictions in our Terms of Service that would interfere with conducting security research

We will not pursue civil action or initiate a complaint to law enforcement for accidental, good-faith violations of this policy.

## Security Best Practices for Users

### Credential Storage

MCP Scooter stores credentials using native OS keychains:
- **Windows:** Windows Credential Manager
- **macOS:** Keychain
- **Linux:** Secret Service (libsecret)

### API Keys

- Never commit API keys to version control
- Use environment variables or the built-in credential manager
- Rotate keys regularly
- Use the minimum required permissions/scopes

### Profile Isolation

- Use separate profiles for work and personal use
- Don't share profile exports that contain sensitive configurations
- Review tool permissions before enabling

## Known Security Considerations

### Local Network Access

MCP Scooter runs a local HTTP server (default port 6277). This server:
- Only binds to localhost by default
- Does not expose services to the network
- Uses CORS headers to restrict access

### Third-Party MCP Servers

When using third-party MCP servers:
- Review the source code when possible
- Only install from trusted sources
- Be cautious with tools that request broad permissions

---

*MCP Scooter is maintained by [Balacode.io](https://balacode.io)*
