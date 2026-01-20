#!/usr/bin/env node
/**
 * MCP Protocol Logger
 * Wraps any stdio MCP server and logs all JSON-RPC messages
 * 
 * Usage: node mcp-logger.js <command> [args...]
 * Example: node mcp-logger.js docker mcp gateway run
 */

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

const logFile = path.join(__dirname, 'mcp-protocol-log.jsonl');

// Clear previous log
fs.writeFileSync(logFile, '');

function log(direction, data) {
    const entry = {
        timestamp: new Date().toISOString(),
        direction,
        data: data.toString().trim()
    };
    
    // Try to parse as JSON for prettier logging
    try {
        entry.parsed = JSON.parse(entry.data);
    } catch (e) {
        // Not JSON, keep as string
    }
    
    fs.appendFileSync(logFile, JSON.stringify(entry) + '\n');
    
    // Also log to stderr for real-time visibility
    process.stderr.write(`[${direction}] ${entry.data.substring(0, 200)}${entry.data.length > 200 ? '...' : ''}\n`);
}

// Get command and args from CLI
const [,, command, ...args] = process.argv;

if (!command) {
    console.error('Usage: node mcp-logger.js <command> [args...]');
    process.exit(1);
}

console.error(`[MCP Logger] Starting: ${command} ${args.join(' ')}`);
console.error(`[MCP Logger] Log file: ${logFile}`);

// Spawn the actual MCP server
// Pass through all environment variables to ensure Docker MCP works correctly
const child = spawn(command, args, {
    stdio: ['pipe', 'pipe', 'pipe'], // pipe all streams so we can log stderr too
    shell: false, // Don't use shell - let Node find the executable
    windowsHide: true,
    env: process.env // Explicitly pass all environment variables
});

// Forward stdin to child, logging each message
process.stdin.on('data', (data) => {
    log('CLIENT->SERVER', data);
    child.stdin.write(data);
});

process.stdin.on('end', () => {
    child.stdin.end();
});

// Forward child stdout to our stdout, logging each message
child.stdout.on('data', (data) => {
    log('SERVER->CLIENT', data);
    process.stdout.write(data);
});

// Log stderr but don't forward it (it would corrupt the JSON-RPC stream)
child.stderr.on('data', (data) => {
    const stderrStr = data.toString();
    // Log to our log file
    const entry = {
        timestamp: new Date().toISOString(),
        direction: 'SERVER-STDERR',
        data: stderrStr.trim()
    };
    fs.appendFileSync(logFile, JSON.stringify(entry) + '\n');
    // Also show on our stderr for debugging
    process.stderr.write(stderrStr);
});

child.on('error', (err) => {
    console.error(`[MCP Logger] Error: ${err.message}`);
    process.exit(1);
});

child.on('exit', (code) => {
    console.error(`[MCP Logger] Process exited with code ${code}`);
    process.exit(code || 0);
});

// Handle our own termination
process.on('SIGINT', () => {
    child.kill('SIGINT');
});

process.on('SIGTERM', () => {
    child.kill('SIGTERM');
});
