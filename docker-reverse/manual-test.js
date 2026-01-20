#!/usr/bin/env node
/**
 * Manual MCP Protocol Tester
 * Sends JSON-RPC messages to Docker MCP gateway and logs responses
 */

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const readline = require('readline');

const logFile = path.join(__dirname, 'manual-test-log.jsonl');

// Clear previous log
fs.writeFileSync(logFile, '');

function log(direction, data) {
    const entry = {
        timestamp: new Date().toISOString(),
        direction,
        data: data
    };
    
    // Try to parse as JSON for prettier logging
    try {
        if (typeof data === 'string') {
            entry.parsed = JSON.parse(data);
        }
    } catch (e) {
        // Not JSON, keep as string
    }
    
    fs.appendFileSync(logFile, JSON.stringify(entry, null, 2) + '\n---\n');
    console.log(`\n[${direction}]`, typeof data === 'string' ? data.substring(0, 500) : JSON.stringify(data, null, 2).substring(0, 500));
}

console.log('Starting Docker MCP Gateway...');
console.log('Log file:', logFile);

// Spawn Docker MCP gateway
const child = spawn('docker', ['mcp', 'gateway', 'run'], {
    stdio: ['pipe', 'pipe', 'pipe'],
    env: process.env
});

let buffer = '';

// Handle stdout (JSON-RPC responses)
child.stdout.on('data', (data) => {
    buffer += data.toString();
    
    // Try to parse complete JSON messages (newline-delimited)
    const lines = buffer.split('\n');
    buffer = lines.pop() || ''; // Keep incomplete line in buffer
    
    for (const line of lines) {
        if (line.trim()) {
            log('SERVER->CLIENT', line.trim());
        }
    }
});

// Handle stderr (Docker MCP logs)
child.stderr.on('data', (data) => {
    const str = data.toString();
    // Only log important stderr messages
    if (str.includes('error') || str.includes('panic') || str.includes('Initialized') || str.includes('Start stdio')) {
        console.error('[STDERR]', str.trim());
    }
});

child.on('error', (err) => {
    console.error('Process error:', err.message);
});

child.on('exit', (code) => {
    console.log('Process exited with code:', code);
    process.exit(code || 0);
});

// Wait for gateway to initialize, then send messages
let messageId = 0;

function sendMessage(method, params = {}) {
    const msg = {
        jsonrpc: '2.0',
        id: ++messageId,
        method,
        params
    };
    const json = JSON.stringify(msg);
    log('CLIENT->SERVER', json);
    child.stdin.write(json + '\n');
    return messageId;
}

function sendNotification(method, params = {}) {
    const msg = {
        jsonrpc: '2.0',
        method,
        params
    };
    const json = JSON.stringify(msg);
    log('CLIENT->SERVER (notification)', json);
    child.stdin.write(json + '\n');
}

// Sequence of messages to send
async function runTest() {
    console.log('\n--- Waiting for gateway to initialize (15s) ---');
    await new Promise(r => setTimeout(r, 15000));
    
    console.log('\n--- Step 1: Initialize ---');
    sendMessage('initialize', {
        protocolVersion: '2024-11-05',
        capabilities: {},
        clientInfo: { name: 'manual-test', version: '1.0.0' }
    });
    await new Promise(r => setTimeout(r, 2000));
    
    console.log('\n--- Step 2: Send initialized notification ---');
    sendNotification('notifications/initialized');
    await new Promise(r => setTimeout(r, 1000));
    
    console.log('\n--- Step 3: List tools (before adding brave) ---');
    sendMessage('tools/list');
    await new Promise(r => setTimeout(r, 2000));
    
    console.log('\n--- Step 4: Call docker_mcp_list_servers ---');
    sendMessage('tools/call', {
        name: 'docker_mcp_list_servers',
        arguments: {}
    });
    await new Promise(r => setTimeout(r, 2000));
    
    console.log('\n--- Step 5: Call docker_mcp_add_server to add brave ---');
    sendMessage('tools/call', {
        name: 'docker_mcp_add_server',
        arguments: { server_name: 'brave' }
    });
    await new Promise(r => setTimeout(r, 5000)); // Give it time to start the server
    
    console.log('\n--- Step 6: List tools (after adding brave) ---');
    sendMessage('tools/list');
    await new Promise(r => setTimeout(r, 2000));
    
    console.log('\n--- Step 7: Call brave_web_search ---');
    sendMessage('tools/call', {
        name: 'brave_web_search',
        arguments: { query: 'AI news January 2026' }
    });
    await new Promise(r => setTimeout(r, 10000)); // Give search time to complete
    
    console.log('\n--- Test complete! Check manual-test-log.jsonl for full output ---');
    console.log('Closing in 3 seconds...');
    await new Promise(r => setTimeout(r, 3000));
    
    child.stdin.end();
}

runTest().catch(err => {
    console.error('Test failed:', err);
    process.exit(1);
});
