#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');

// Determine binary name based on platform
const binaryName = process.platform === 'win32' ? 'starknet-faucet.exe' : 'starknet-faucet';
const binaryPath = path.join(__dirname, binaryName);

// Execute the binary with all arguments
const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env
});

child.on('exit', (code) => {
  process.exit(code || 0);
});

child.on('error', (err) => {
  console.error('Failed to start binary:', err.message);
  process.exit(1);
});
