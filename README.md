# 🚰 Starknet Faucet

Get testnet **ETH** and **STRK** tokens for Starknet Sepolia - directly from your terminal!

[![npm](https://img.shields.io/npm/v/starknet-faucet)](https://www.npmjs.com/package/starknet-faucet)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Downloads](https://img.shields.io/npm/dm/starknet-faucet)](https://www.npmjs.com/package/starknet-faucet)

## ✨ Features

- 🖥️ **Terminal-First**: No browser needed, perfect for development workflows
- 💎 **Both Tokens**: Get STRK and ETH for Starknet Sepolia testnet
- 🤖 **Smart Protection**: CAPTCHA + Proof of Work prevents abuse
- ⚡ **Fast**: Tokens arrive in ~30 seconds
- 🎨 **Beautiful CLI**: Colored output with progress indicators
- 🔗 **Explorer Links**: Direct links to view your transaction

## 📦 Installation

Install globally via npm:

```bash
npm install -g starknet-faucet
```

That's it! The CLI will be available as `starknet-faucet` command.

## 🚀 Quick Start

### Request STRK Tokens

```bash
starknet-faucet request 0xYOUR_ADDRESS
```

### Request ETH Tokens

```bash
starknet-faucet request 0xYOUR_ADDRESS --token ETH
```

### Request Both ETH and STRK

```bash
starknet-faucet request 0xYOUR_ADDRESS --both
```

## 📖 Commands

### Request Tokens
```bash
starknet-faucet request <ADDRESS> [flags]
```

**Flags:**
- `--token string` - Token to request: `ETH` or `STRK` (default: `STRK`)
- `--both` - Request both ETH and STRK tokens
- `--json` - Output result in JSON format

**Example:**
```bash
$ starknet-faucet request 0x0223C87c0641e802a7DA24E68a46F8b0094F17762bf703284Bba99A7e62970D4

   ███████╗████████╗ █████╗ ██████╗ ██╗  ██╗███╗   ██╗███████╗████████╗    ███████╗ █████╗ ██╗   ██╗ ██████╗███████╗████████╗
   ██╔════╝╚══██╔══╝██╔══██╗██╔══██╗██║ ██╔╝████╗  ██║██╔════╝╚══██╔══╝    ██╔════╝██╔══██╗██║   ██║██╔════╝██╔════╝╚══██╔══╝
   ███████╗   ██║   ███████║██████╔╝█████╔╝ ██╔██╗ ██║█████╗     ██║       █████╗  ███████║██║   ██║██║     █████╗     ██║
   ╚════██║   ██║   ██╔══██║██╔══██╗██╔═██╗ ██║╚██╗██║██╔══╝     ██║       ██╔══╝  ██╔══██║██║   ██║██║     ██╔══╝     ██║
   ███████║   ██║   ██║  ██║██║  ██║██║  ██╗██║ ╚████║███████╗   ██║       ██║     ██║  ██║╚██████╔╝╚██████╗███████╗   ██║
   ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝       ╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚═════╝╚══════╝   ╚═╝

                                         ⚡ Powered by Starknet • Secured by PoW ⚡

═══════════════════════════════════════════════════════
  🤖 Quick Verification (helps prevent bot abuse)
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

### Check Status
Check if an address is in cooldown:
```bash
starknet-faucet status <ADDRESS>
```

### Get Faucet Info
View faucet balance and limits:
```bash
starknet-faucet info
```

## 💰 Distribution Limits

| Token | Amount per Request | Rate Limit |
|-------|-------------------|------------|
| STRK  | 10 STRK          | 2/hour, 5/day per address |
| ETH   | 0.02 ETH         | 2/hour, 5/day per address |

**Additional Protection:**
- Maximum 10 requests per hour per IP
- Maximum 20 requests per day per IP
- Proof of Work challenge (difficulty: 4)
- Interactive CAPTCHA verification

## 🔧 Advanced Usage

### Custom API URL
```bash
starknet-faucet request 0xADDRESS --api-url https://your-faucet-instance.com
```

### JSON Output
```bash
starknet-faucet request 0xADDRESS --json
```

### Verbose Logging
```bash
starknet-faucet request 0xADDRESS --verbose
```

## 🏥 Health Check

To check if the faucet API is running:

```bash
curl https://starknet-faucet-gnq5.onrender.com/api/v1/info
```

**Response:**
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

## 📦 NPM Package

**Package:** [@npmjs.com/package/starknet-faucet](https://www.npmjs.com/package/starknet-faucet)

The npm package automatically downloads the appropriate pre-built binary for your platform:
- Linux (AMD64, ARM64)
- macOS (Intel, Apple Silicon)
- Windows (AMD64)

## 🔐 Security Features

- **Proof of Work**: CPU-based challenge prevents automated abuse
- **CAPTCHA Questions**: Human verification with 100+ questions
- **Rate Limiting**: Per-IP and per-address limits
- **Challenge Expiration**: 5-minute TTL on PoW challenges
- **Balance Protection**: Stops distribution at 5% remaining balance

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [starknet.go](https://github.com/NethermindEth/starknet.go) v0.17.0
- Powered by [Starknet](https://starknet.io/)
- Block explorer by [Voyager](https://voyager.online/)

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/Giri-Aayush/starknet-faucet/issues)
- **NPM**: [npm package](https://www.npmjs.com/package/starknet-faucet)
- **Faucet API**: https://starknet-faucet-gnq5.onrender.com
- **Developer**: [Aayush Giri](https://github.com/Giri-Aayush)

---

Made with ❤️ for the Starknet community
