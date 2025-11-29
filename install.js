#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const { promisify } = require('util');
const stream = require('stream');

const pipeline = promisify(stream.pipeline);

const VERSION = 'v1.0.16';
const GITHUB_REPO = 'Giri-Aayush/starknet-faucet';

// Detect platform and architecture
function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;

  let binaryName = 'starknet-faucet';

  if (platform === 'darwin') {
    if (arch === 'arm64') {
      binaryName += '-macos-arm64';
    } else {
      binaryName += '-macos-amd64';
    }
  } else if (platform === 'linux') {
    if (arch === 'arm64') {
      binaryName += '-linux-arm64';
    } else {
      binaryName += '-linux-amd64';
    }
  } else if (platform === 'win32') {
    binaryName += '-windows-amd64.exe';
  } else {
    throw new Error(`Unsupported platform: ${platform} ${arch}`);
  }

  return binaryName;
}

async function downloadBinary() {
  try {
    const binaryName = getPlatform();
    const downloadUrl = `https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${binaryName}`;
    const binDir = path.join(__dirname, 'bin');
    const binaryPath = path.join(binDir, process.platform === 'win32' ? 'starknet-faucet.exe' : 'starknet-faucet');

    console.log(`📦 Downloading Starknet Faucet CLI ${VERSION}...`);
    console.log(`   Platform: ${process.platform} (${process.arch})`);
    console.log(`   Binary: ${binaryName}`);

    // Ensure bin directory exists
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    // Download binary with progress indicator
    await new Promise((resolve, reject) => {
      https.get(downloadUrl, (response) => {
        if (response.statusCode === 302 || response.statusCode === 301) {
          // Follow redirect
          https.get(response.headers.location, (redirectResponse) => {
            if (redirectResponse.statusCode !== 200) {
              reject(new Error(`Download failed with status ${redirectResponse.statusCode}`));
              return;
            }

            const totalBytes = parseInt(redirectResponse.headers['content-length'], 10);
            let downloadedBytes = 0;
            let lastPercent = 0;

            // Progress indicator
            console.log(`\n   Downloading...`);

            redirectResponse.on('data', (chunk) => {
              downloadedBytes += chunk.length;
              const percent = Math.floor((downloadedBytes / totalBytes) * 100);

              // Update every 10%
              if (percent >= lastPercent + 10 || percent === 100) {
                const bar = '█'.repeat(Math.floor(percent / 5)) + '░'.repeat(20 - Math.floor(percent / 5));
                process.stdout.write(`\r   [${bar}] ${percent}%`);
                lastPercent = percent;
              }
            });

            const fileStream = fs.createWriteStream(binaryPath);
            pipeline(redirectResponse, fileStream)
              .then(() => {
                console.log('\n');
                resolve();
              })
              .catch(reject);
          });
        } else if (response.statusCode === 200) {
          const totalBytes = parseInt(response.headers['content-length'], 10);
          let downloadedBytes = 0;
          let lastPercent = 0;

          // Progress indicator
          console.log(`\n   Downloading...`);

          response.on('data', (chunk) => {
            downloadedBytes += chunk.length;
            const percent = Math.floor((downloadedBytes / totalBytes) * 100);

            // Update every 10%
            if (percent >= lastPercent + 10 || percent === 100) {
              const bar = '█'.repeat(Math.floor(percent / 5)) + '░'.repeat(20 - Math.floor(percent / 5));
              process.stdout.write(`\r   [${bar}] ${percent}%`);
              lastPercent = percent;
            }
          });

          const fileStream = fs.createWriteStream(binaryPath);
          pipeline(response, fileStream)
            .then(() => {
              console.log('\n');
              resolve();
            })
            .catch(reject);
        } else {
          reject(new Error(`Download failed with status ${response.statusCode}. Binary may not exist yet for ${binaryName}.`));
        }
      }).on('error', reject);
    });

    // Make binary executable (Unix systems)
    if (process.platform !== 'win32') {
      fs.chmodSync(binaryPath, 0o755);
    }

    console.log('✓ Installation complete!\n');
    console.log('Run: npx starknet-faucet --help');

  } catch (error) {
    console.error('\n❌ Installation failed:', error.message);
    console.error('\n📝 Manual installation:');
    console.error(`   1. Download binary from: https://github.com/${GITHUB_REPO}/releases`);
    console.error(`   2. Place it in your PATH`);
    console.error(`   3. Run: starknet-faucet --help\n`);
    process.exit(1);
  }
}

downloadBinary();
