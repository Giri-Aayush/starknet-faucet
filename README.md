# 🚰 Starknet Faucet

The first **terminal-based faucet** for Starknet Sepolia testnet. Request testnet ETH and STRK tokens directly from your command line.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Built with starknet.go](https://img.shields.io/badge/Built%20with-starknet.go%20v0.17.0-orange)](https://github.com/NethermindEth/starknet.go)

## ✨ Features

- **CLI-First**: Request tokens directly from your terminal
- **No Browser Needed**: Perfect for development workflows
- **Both ETH & STRK**: Support for both Starknet tokens
- **Proof of Work**: Fair distribution using PoW challenges
- **Rate Limited**: 24-hour cooldown per address
- **Voyager Integration**: Direct links to transaction explorer
- **Beautiful UI**: Colored output with progress indicators

## 📦 Installation

### Option 1: Install Script (Recommended)

```bash
curl -L https://raw.githubusercontent.com/aayushgiri/starknet-faucet/main/scripts/install.sh | sh
```

### Option 2: Go Install

```bash
go install github.com/Giri-Aayush/starknet-faucet/cmd/cli@latest
```

### Option 3: Download Binary

Download the latest binary from [GitHub Releases](https://github.com/Giri-Aayush/starknet-faucet/releases).

### Option 4: Build from Source

```bash
git clone https://github.com/Giri-Aayush/starknet-faucet.git
cd starknet-faucet
go build -o starknet-faucet ./cmd/cli
sudo mv starknet-faucet /usr/local/bin/
```

## 🚀 Quick Start

### Request STRK Tokens

```bash
starknet-faucet request 0x0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795f --token STRK
```

### Request ETH Tokens

```bash
starknet-faucet request 0x0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795f --token ETH
```

### Request Both Tokens

```bash
starknet-faucet request 0x0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795f --both
```

### Check Address Status

```bash
starknet-faucet status 0x0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795f
```

### Get Faucet Info

```bash
starknet-faucet info
```

## 📖 Usage

```
starknet-faucet [command] [flags]

Commands:
  request     Request testnet tokens
  status      Check cooldown status of an address
  info        Get faucet information
  help        Help about any command

Flags:
  --api-url string   Faucet API URL (default "https://starknet-faucet-gnq5.onrender.com")
  --json             Output in JSON format
  -v, --verbose      Verbose output
  --help             Help for starknet-faucet
```

## 💰 Distribution Limits

| Token | Amount per Request | Cooldown Period |
|-------|-------------------|-----------------|
| STRK  | 100 STRK          | 24 hours        |
| ETH   | 0.02 ETH          | 24 hours        |

**Rate Limits:**
- 5 requests per hour per IP
- 3 requests per day per IP

## 🔒 Security

The faucet implements multiple layers of protection:

1. **Proof of Work**: Clients must solve a SHA256 challenge (difficulty: 4)
2. **Address Cooldown**: 24-hour cooldown period per address
3. **IP Rate Limiting**: Hourly and daily limits per IP
4. **Challenge TTL**: Challenges expire after 5 minutes
5. **No Challenge Reuse**: Each challenge can only be used once

## 🏗️ Architecture

```
┌─────────────┐
│  CLI Tool   │
└──────┬──────┘
       │ HTTP/REST
       ▼
┌─────────────┐      ┌─────────────┐
│  API Server │◄────►│    Redis    │
└──────┬──────┘      └─────────────┘
       │
       │ starknet.go
       ▼
┌─────────────┐
│  Starknet   │
│   Sepolia   │
└─────────────┘
```

## 🛠️ Development

### Prerequisites

- Go 1.22+
- Redis
- Starknet account with testnet tokens

### Setup

1. Clone the repository:
```bash
git clone https://github.com/Giri-Aayush/starknet-faucet.git
cd starknet-faucet
```

2. Install dependencies:
```bash
go mod download
```

3. Copy environment variables:
```bash
cp .env.example .env
```

4. Configure your `.env` file:
```env
FAUCET_PRIVATE_KEY=0x...
FAUCET_ADDRESS=0x...
STARKNET_RPC_URL=https://starknet-sepolia.public.blastapi.io
REDIS_URL=redis://localhost:6379
```

### Run Server Locally

```bash
# Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# Run server
go run cmd/server/main.go
```

### Run CLI Locally

```bash
# Build CLI
go build -o starknet-faucet ./cmd/cli

# Test CLI
./starknet-faucet request 0x... --api-url http://localhost:3000
```

### Run with Docker Compose

```bash
docker-compose -f deployments/docker-compose.yml up
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/pow/...
```

## 📝 API Documentation

### Endpoints

#### POST `/api/v1/challenge`
Get a new PoW challenge.

**Response:**
```json
{
  "challenge_id": "a1b2c3d4...",
  "challenge": "d4f5e6a7b8c9...",
  "difficulty": 4
}
```

#### POST `/api/v1/faucet`
Request tokens from the faucet.

**Request:**
```json
{
  "address": "0x0742...",
  "token": "STRK",
  "challenge_id": "a1b2c3d4...",
  "nonce": 42731
}
```

**Response:**
```json
{
  "success": true,
  "tx_hash": "0x07a1b2c3d4e5f6...",
  "amount": "100",
  "token": "STRK",
  "explorer_url": "https://sepolia.voyager.online/tx/0x07a1...",
  "message": "Tokens sent successfully"
}
```

#### GET `/api/v1/status/:address`
Check address cooldown status.

**Response:**
```json
{
  "address": "0x0742...",
  "can_request": false,
  "last_request": "2025-11-11T14:30:00Z",
  "next_request_time": "2025-11-12T14:30:00Z",
  "remaining_hours": 23.5
}
```

#### GET `/api/v1/info`
Get faucet information.

**Response:**
```json
{
  "network": "sepolia",
  "limits": {
    "strk_per_request": "100",
    "eth_per_request": "0.02",
    "cooldown_hours": 24
  },
  "pow": {
    "enabled": true,
    "difficulty": 4
  },
  "faucet_balance": {
    "strk": "50000",
    "eth": "10"
  }
}
```

## 🚀 Deployment

### Railway / Render

1. Fork this repository
2. Connect to Railway/Render
3. Set environment variables
4. Deploy!

### VPS Deployment

```bash
# Clone and build
git clone https://github.com/Giri-Aayush/starknet-faucet.git
cd starknet-faucet

# Configure environment
cp .env.example .env
nano .env

# Run with Docker Compose
docker-compose -f deployments/docker-compose.yml up -d
```

### Kubernetes

```bash
kubectl apply -f deployments/kubernetes/
```

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [starknet.go](https://github.com/NethermindEth/starknet.go) v0.17.0
- Powered by [Starknet](https://starknet.io/)
- Block explorer by [Voyager](https://voyager.online/)

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/Giri-Aayush/starknet-faucet/issues)

## 🗺️ Roadmap

- [ ] GitHub OAuth for higher limits
- [ ] Discord bot integration
- [ ] Multi-network support (mainnet)
- [ ] Web dashboard
- [ ] API keys for projects
- [ ] Auto-refill monitoring

---

**Made with ❤️ for the Starknet community**
