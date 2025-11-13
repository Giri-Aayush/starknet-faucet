# Starknet Faucet

A command-line tool for requesting testnet tokens (ETH and STRK) on Starknet Sepolia. Built to simplify the developer experience when working with Starknet applications.

[![npm](https://img.shields.io/npm/v/starknet-faucet)](https://www.npmjs.com/package/starknet-faucet)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Downloads](https://img.shields.io/npm/dm/starknet-faucet)](https://www.npmjs.com/package/starknet-faucet)

## Why This Exists

While building on Starknet, I found myself constantly switching between terminal and browser to get testnet tokens. It disrupted my workflow and slowed down development. This tool brings the faucet directly to your terminal, where development happens.

[Read more about the motivation behind this project](https://x.com/AayushStack/status/1989022633657340373?s=20)

## Features

- **Terminal-native**: Request tokens without leaving your development environment
- **Dual token support**: Get both STRK and ETH for Starknet Sepolia
- **Abuse protection**: Proof of Work and CAPTCHA verification prevent bot abuse
- **Fast delivery**: Tokens arrive in approximately 30 seconds
- **Transaction tracking**: Direct explorer links to monitor your requests
- **Cross-platform**: Works on Linux, macOS, and Windows

## Installation

```bash
npm install -g starknet-faucet
```

The CLI will be available as the `starknet-faucet` command.

## Usage

### Request STRK tokens (default)
```bash
starknet-faucet request 0xYOUR_ADDRESS
```

### Request ETH tokens
```bash
starknet-faucet request 0xYOUR_ADDRESS --token ETH
```

### Request both tokens
```bash
starknet-faucet request 0xYOUR_ADDRESS --both
```

### Check address status
```bash
starknet-faucet status 0xYOUR_ADDRESS
```

### View faucet information
```bash
starknet-faucet info
```

## Commands

### request
Request tokens for a Starknet address.

**Flags:**
- `--token string` - Token type: `ETH` or `STRK` (default: `STRK`)
- `--both` - Request both ETH and STRK tokens
- `--json` - Output in JSON format
- `--verbose, -v` - Enable verbose logging
- `--api-url string` - Custom faucet API URL

**Example output:**
```bash
$ starknet-faucet request 0x0223C87c0641e802a7DA24E68a46F8b0094F17762bf703284Bba99A7e62970D4

   ███████╗████████╗ █████╗ ██████╗ ██╗  ██╗███╗   ██╗███████╗████████╗    ███████╗ █████╗ ██╗   ██╗ ██████╗███████╗████████╗
   ██╔════╝╚══██╔══╝██╔══██╗██╔══██╗██║ ██╔╝████╗  ██║██╔════╝╚══██╔══╝    ██╔════╝██╔══██╗██║   ██║██╔════╝██╔════╝╚══██╔══╝
   ███████╗   ██║   ███████║██████╔╝█████╔╝ ██╔██╗ ██║█████╗     ██║       █████╗  ███████║██║   ██║██║     █████╗     ██║
   ╚════██║   ██║   ██╔══██║██╔══██╗██╔═██╗ ██║╚██╗██║██╔══╝     ██║       ██╔══╝  ██╔══██║██║   ██║██║     ██╔══╝     ██║
   ███████║   ██║   ██║  ██║██║  ██║██║  ██╗██║ ╚████║███████╗   ██║       ██║     ██║  ██║╚██████╔╝╚██████╗███████╗   ██║
   ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝       ╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚═════╝╚══════╝   ╚═╝

                                         Made with love • Secured by PoW

═══════════════════════════════════════════════════════
  Quick Verification (helps prevent bot abuse)
═══════════════════════════════════════════════════════

  What is 2 + 2? 4

  ✓ Correct!

→ Requesting STRK for 0x0223C87c0641e802a7DA24E68a46F8b0094F17762bf703284Bba99A7e62970D4

✓ Challenge received
✓ Challenge solved in 0.1s (nonce: 277112)
✓ Transaction submitted!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Amount:  10 STRK
  TX Hash: 0x469d165e...e01efede

  🔗 https://sepolia.voyager.online/tx/0x469d165e06ac3f1de87286ca240b13d81d3eab9b65f209f9720f853e01efede
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✓ Tokens will arrive in ~30 seconds.
```

### status
Check if an address is in cooldown period.

```bash
starknet-faucet status 0xYOUR_ADDRESS
```

### info
View faucet balance, distribution limits, and configuration.

```bash
starknet-faucet info
```

## Distribution Limits

| Token | Amount per Request | Cooldown Period |
|-------|-------------------|-----------------|
| STRK  | 10 STRK          | 24 hours per address |
| ETH   | 0.02 ETH         | 24 hours per address |

**Rate limiting:**
- IP-based limits: 10 requests/hour, 20 requests/day
- Address-based limits: 2 requests/hour, 5 requests/day

## Security

The faucet implements multiple layers of protection:

- **Proof of Work**: CPU-based challenge with difficulty 4
- **CAPTCHA verification**: Human verification through interactive questions
- **Rate limiting**: Both IP-based and address-based limits
- **Challenge expiration**: 5-minute time-to-live on PoW challenges
- **Balance protection**: Automatic shutdown at 5% remaining balance

## API Health Check

To verify the faucet API is operational:

```bash
curl https://starknet-faucet-gnq5.onrender.com/api/v1/info
```

**Sample response:**
```json
{
  "network": "sepolia",
  "limits": {
    "strk_per_request": "10",
    "eth_per_request": "0.02",
    "cooldown_hours": 24
  },
  "pow": {
    "enabled": true,
    "difficulty": 4
  },
  "faucet_balance": {
    "strk": "79.99",
    "eth": "0.05"
  }
}
```

## Platform Support

Pre-built binaries are available for:
- Linux (AMD64, ARM64)
- macOS (Intel, Apple Silicon)
- Windows (AMD64)

The npm package automatically downloads the correct binary for your platform during installation.

## Technical Details

- Built with Go 1.23+
- Uses [starknet.go](https://github.com/NethermindEth/starknet.go) v0.17.0
- Backend API hosted on Render
- Redis-based caching for rate limiting
- Transaction tracking via [Voyager](https://voyager.online/)

## Contributing

Contributions are welcome. Please submit pull requests or open issues on GitHub.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Links

- **npm package**: https://www.npmjs.com/package/starknet-faucet
- **GitHub repository**: https://github.com/Giri-Aayush/starknet-faucet
- **API endpoint**: https://starknet-faucet-gnq5.onrender.com
- **Developer**: [Aayush Giri](https://github.com/Giri-Aayush)

---

Built for developers who prefer staying in the terminal.
