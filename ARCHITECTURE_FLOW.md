# Starknet Faucet - Complete Architecture & Flow

## Overview

Your faucet has **3 components** that work together:

1. **CLI (Client)** - Users install via npm and run on their terminal
2. **Backend API (Server)** - Deployed on Koyeb, handles requests and sends transactions
3. **Frontend (Website)** - Shows stats and provides web-based interface

---

## Component Details

### 1. CLI (Command-Line Tool)

**Location:** `cmd/cli/main.go`, `pkg/cli/*`

**What it does:**
- Runs locally on user's machine
- Shows banner, CAPTCHA, progress
- Solves Proof of Work locally
- Sends HTTP requests to Backend API

**Installation Flow:**
```
User runs: npm install -g starknet-faucet@1.0.16
    ↓
npm executes: node install.js
    ↓
install.js downloads binary from GitHub releases:
  https://github.com/Giri-Aayush/starknet-faucet/releases/download/v1.0.16/starknet-faucet-linux-amd64
    ↓
Binary saved to: node_modules/starknet-faucet/bin/starknet-faucet
    ↓
Made executable: chmod +x
    ↓
npm creates global symlink: /usr/local/bin/starknet-faucet → node_modules/...
```

**Runtime Flow:**
```
User runs: starknet-faucet request 0x123...
    ↓
CLI binary executes (Go program)
    ↓
Shows banner + CAPTCHA
    ↓
Makes HTTP requests to Backend API
```

### 2. Backend API (Server)

**Location:** `cmd/server/main.go`, `internal/*`

**Deployed on:** Koyeb (https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app)

**What it does:**
- Receives HTTP requests from CLI and Frontend
- Validates PoW solutions
- Manages rate limiting (Redis)
- Sends blockchain transactions (via Alchemy RPC)
- Returns transaction hashes

**Deployment:**
```
Code pushed to GitHub
    ↓
Koyeb auto-deploys (watches main branch)
    ↓
Builds Docker image: deployments/docker/Dockerfile.server
    ↓
Starts container on Koyeb infrastructure
    ↓
Exposed at: https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app
```

**Running Services:**
- Fiber web server (port 8080)
- Connected to Upstash Redis (rate limiting)
- Connected to Alchemy Starknet RPC (blockchain)

### 3. Frontend (Website)

**What it shows:**
- Live faucet balance
- Distribution limits
- Request statistics
- Health status

**How it works:**
Makes HTTP GET requests to Backend API endpoints:
- `/health` - Server status
- `/api/v1/info` - Faucet info and balance
- (Other endpoints for web-based token requests)

---

## Complete Request Flow

Let me trace what happens when a user requests tokens:

### Step 1: User Runs CLI Command

```bash
starknet-faucet request 0x02ca67d3b01d9546a995880cc88173cd7335044f222370f047275b90c8e384fb
```

**What happens locally (on user's machine):**

```
┌─────────────────────────────────────┐
│  User's Terminal (CLI Binary)      │
├─────────────────────────────────────┤
│                                     │
│  1. Parse arguments                 │
│     ✓ Address: 0x02ca67...          │
│     ✓ Token: STRK (default)         │
│                                     │
│  2. Show banner                     │
│     ────────────────────────        │
│     Starknet Terminal Faucet        │
│     ────────────────────────        │
│                                     │
│  3. Ask CAPTCHA                     │
│     Q: What drink is made from      │
│        coffee beans?                │
│     User types: coffee              │
│     ✓ Correct!                      │
│                                     │
└─────────────────────────────────────┘
```

**Code:** `pkg/cli/commands/request.go` line 89-101

```go
// Print banner (unless JSON output)
if !jsonOut {
    ui.PrintBanner()

    // Ask verification question (3 attempts)
    correct, err := captcha.AskQuestionWithRetries(3)
    if err != nil {
        return fmt.Errorf("verification failed: %w", err)
    }
}
```

---

### Step 2: Request Challenge from Backend

**CLI → Backend API**

```
┌─────────────────────────────────────┐
│  CLI makes HTTP GET request         │
├─────────────────────────────────────┤
│                                     │
│  GET https://intermediate-          │
│  albertine-aayushgiri-e93ace53.     │
│  koyeb.app/api/v1/challenge         │
│                                     │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  Backend API (Koyeb)                │
├─────────────────────────────────────┤
│                                     │
│  Handler: GetChallenge()            │
│                                     │
│  1. Generate random challenge       │
│     challenge = random hex string   │
│                                     │
│  2. Store in Redis (5 min TTL)      │
│     challengeID → challenge         │
│                                     │
│  3. Return JSON response            │
│                                     │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  Response to CLI                    │
├─────────────────────────────────────┤
│  {                                  │
│    "challenge_id": "abc123",        │
│    "challenge": "0x4f8a2...",       │
│    "difficulty": 6                  │
│  }                                  │
└─────────────────────────────────────┘
```

**Backend Code:** `internal/api/handlers/challenge.go`

```go
func (h *Handler) GetChallenge(c *fiber.Ctx) error {
    // Generate random challenge
    challenge := generateRandomChallenge()
    challengeID := generateID()

    // Store in Redis with 5 min expiry
    h.redis.Set(ctx, "challenge:"+challengeID, challenge, 5*time.Minute)

    return c.JSON(ChallengeResponse{
        ChallengeID: challengeID,
        Challenge:   challenge,
        Difficulty:  h.config.PoWDifficulty, // 6
    })
}
```

**CLI Code:** `pkg/cli/api_client.go`

```go
func (c *APIClient) GetChallenge() (*models.ChallengeResponse, error) {
    resp, err := http.Get(c.baseURL + "/api/v1/challenge")
    // ... parse JSON response
}
```

---

### Step 3: Solve Proof of Work (Locally on User's Machine)

**What happens:**

```
┌─────────────────────────────────────┐
│  CLI - PoW Solver                   │
├─────────────────────────────────────┤
│                                     │
│  Challenge: 0x4f8a2...              │
│  Difficulty: 6                      │
│                                     │
│  Finding nonce where:               │
│  SHA256(challenge + nonce)          │
│  starts with 6 zeros                │
│                                     │
│  Try nonce = 0:                     │
│    hash = 0x8a3f... ✗ (only 2)     │
│                                     │
│  Try nonce = 1:                     │
│    hash = 0x5b2c... ✗ (only 3)     │
│                                     │
│  Try nonce = 800047:                │
│    hash = 0x000000a1... ✓ (6!)     │
│                                     │
│  ✓ Challenge solved in 0.2s         │
│    Nonce: 800047                    │
│                                     │
└─────────────────────────────────────┘
```

**Why locally?**
- Prevents DDoS attacks (attacker's CPU does work, not your server)
- User sees progress in terminal
- Server just validates the solution

**CLI Code:** `pkg/cli/pow/solver.go`

```go
func (s *Solver) Solve(challenge string, difficulty int, progressCallback func(int64, time.Duration)) (*Result, error) {
    target := strings.Repeat("0", difficulty)

    for nonce := int64(0); ; nonce++ {
        hash := sha256Hash(challenge + fmt.Sprint(nonce))

        if strings.HasPrefix(hash, target) {
            // Found valid nonce!
            return &Result{
                Nonce:    nonce,
                Hash:     hash,
                Duration: time.Since(start),
            }, nil
        }

        // Update progress every 1000 attempts
        if nonce%1000 == 0 && progressCallback != nil {
            progressCallback(nonce, time.Since(start))
        }
    }
}
```

---

### Step 4: Submit Token Request to Backend

**CLI → Backend API**

```
┌─────────────────────────────────────┐
│  CLI makes HTTP POST request        │
├─────────────────────────────────────┤
│                                     │
│  POST https://intermediate-         │
│  albertine-aayushgiri-e93ace53.     │
│  koyeb.app/api/v1/request           │
│                                     │
│  Body (JSON):                       │
│  {                                  │
│    "address": "0x02ca67...",        │
│    "token": "STRK",                 │
│    "challenge_id": "abc123",        │
│    "nonce": 800047                  │
│  }                                  │
│                                     │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  Backend API (Koyeb)                │
├─────────────────────────────────────┤
│                                     │
│  Handler: RequestTokens()           │
│                                     │
│  1. Get client IP                   │
│     ip = "203.0.113.45"             │
│                                     │
│  2. Check rate limits (Redis)       │
│     ✓ IP has 4/5 requests left      │
│     ✓ STRK throttle OK (>1hr ago)   │
│                                     │
│  3. Verify PoW solution             │
│     challenge = Redis.get(abc123)   │
│     hash = SHA256(challenge+800047) │
│     ✓ Starts with 6 zeros           │
│                                     │
│  4. Send blockchain transaction     │
│     (see Step 5)                    │
│                                     │
└─────────────────────────────────────┘
```

**Backend Code:** `internal/api/handlers/request.go`

```go
func (h *Handler) RequestTokens(c *fiber.Ctx) error {
    var req RequestBody
    c.BodyParser(&req)

    // 1. Get IP
    ip := c.IP()

    // 2. Check rate limits
    if err := h.rateLimit.Check(ip, req.Token); err != nil {
        return c.Status(429).JSON(fiber.Map{"error": "rate limit exceeded"})
    }

    // 3. Verify PoW
    challenge, _ := h.redis.Get("challenge:" + req.ChallengeID)
    if !verifyPoW(challenge, req.Nonce, h.difficulty) {
        return c.Status(400).JSON(fiber.Map{"error": "invalid PoW"})
    }

    // 4. Send transaction (see next step)
    txHash, err := h.starknet.Transfer(req.Address, req.Token, amount)

    // 5. Update rate limits in Redis
    h.rateLimit.Record(ip, req.Token)

    return c.JSON(fiber.Map{
        "success": true,
        "tx_hash": txHash,
        "amount": "10",
        "token": "STRK",
    })
}
```

---

### Step 5: Backend Sends Blockchain Transaction

**Backend API → Alchemy RPC → Starknet**

```
┌─────────────────────────────────────┐
│  Backend API                        │
├─────────────────────────────────────┤
│                                     │
│  1. Build transfer transaction      │
│     From: Faucet wallet             │
│           0x31167a4...              │
│     To:   User address              │
│           0x02ca67...               │
│     Amount: 10 STRK                 │
│                                     │
│  2. Sign with faucet private key    │
│     (from env: FAUCET_PRIVATE_KEY)  │
│                                     │
│  3. Send to Starknet via Alchemy    │
│                                     │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  Alchemy RPC                        │
│  (Starknet Gateway)                 │
├─────────────────────────────────────┤
│                                     │
│  POST https://starknet-sepolia.     │
│  g.alchemy.com/starknet/version/    │
│  rpc/v0_9/cWsqpE1AYKEJM6-JaS28G     │
│                                     │
│  Method: starknet_addInvokeTransaction│
│                                     │
│  ✓ Transaction accepted             │
│  TX Hash: 0x6139cd4b...             │
│                                     │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  Starknet Blockchain                │
├─────────────────────────────────────┤
│                                     │
│  Transaction pending...             │
│  (takes ~10-30 seconds)             │
│                                     │
│  ✓ Transaction confirmed            │
│  Block: #123456                     │
│                                     │
└─────────────────────────────────────┘
```

**Backend Code:** `internal/starknet/client.go`

```go
func (fc *FaucetClient) Transfer(toAddress, token string, amount *big.Int) (string, error) {
    // 1. Build transaction calldata
    calldata := buildTransferCalldata(toAddress, amount)

    // 2. Create invoke transaction
    tx := rpc.InvokeTxnV1{
        SenderAddress: fc.accountAddress,
        Calldata:     calldata,
        // ... signatures, nonce, etc.
    }

    // 3. Sign transaction
    signature := fc.account.Sign(tx)

    // 4. Send to Alchemy RPC
    resp, err := fc.provider.AddInvokeTransaction(ctx, tx)

    return resp.TransactionHash, nil
}
```

**Environment Variables (Koyeb):**
```
STARKNET_RPC_URL=https://starknet-sepolia.g.alchemy.com/starknet/version/rpc/v0_9/cWsqpE1AYKEJM6-JaS28G
FAUCET_PRIVATE_KEY=0x5f7c...  (your faucet wallet private key)
FAUCET_ADDRESS=0x31167a4...
```

---

### Step 6: Response Back to User

**Backend API → CLI → User**

```
┌─────────────────────────────────────┐
│  Backend sends JSON response        │
├─────────────────────────────────────┤
│  {                                  │
│    "success": true,                 │
│    "tx_hash": "0x6139cd4b...",      │
│    "amount": "10",                  │
│    "token": "STRK",                 │
│    "explorer_url": "https://        │
│      sepolia.voyager.online/tx/..." │
│  }                                  │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│  CLI receives response              │
├─────────────────────────────────────┤
│                                     │
│  ✓ Transaction submitted!           │
│                                     │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━    │
│    Amount:  10 STRK                 │
│    TX Hash:  0x6139cd4b...82e8d55f  │
│                                     │
│    🔗 https://sepolia.voyager...    │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━    │
│                                     │
│  ✓ Tokens will arrive in ~30s.      │
│                                     │
└─────────────────────────────────────┘
```

---

## Frontend Integration

Your website also connects to the same Backend API:

### Frontend → Backend API

```javascript
// Get faucet info (balance, limits, etc.)
fetch('https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app/api/v1/info')
  .then(res => res.json())
  .then(data => {
    // Update UI with:
    // - STRK balance: data.faucet_balance.strk
    // - ETH balance: data.faucet_balance.eth
    // - Daily limit: data.limits.daily_requests_per_ip
  });

// Check health
fetch('https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app/health')
  .then(res => res.json())
  .then(data => {
    // Show: data.status === "ok"
  });
```

**Backend API Endpoints:**

| Endpoint | Method | Purpose | Used By |
|----------|--------|---------|---------|
| `/health` | GET | Server health check | Frontend, CLI, Monitoring |
| `/api/v1/info` | GET | Faucet info & balance | Frontend |
| `/api/v1/challenge` | GET | Get PoW challenge | CLI |
| `/api/v1/request` | POST | Request tokens | CLI, Frontend |
| `/api/v1/quota` | GET | Check user's quota | CLI |
| `/api/v1/limits` | GET | Get rate limit rules | CLI |

---

## Data Flow Summary

```
┌──────────────┐
│   User's PC  │
│              │
│   CLI Tool   │  (Go binary installed via npm)
└──────┬───────┘
       │ HTTP requests
       ↓
┌──────────────┐
│    Koyeb     │
│              │
│  Backend API │  (Go server, Fiber framework)
└──┬────────┬──┘
   │        │
   │        └─────→ Upstash Redis (rate limiting)
   │
   └────────────→ Alchemy RPC
                  ↓
             Starknet Blockchain
```

**Parallel Flow:**

```
┌──────────────┐
│   Browser    │
│              │
│   Frontend   │  (React/Vue/HTML)
└──────┬───────┘
       │ HTTP requests
       ↓
┌──────────────┐
│    Koyeb     │
│              │
│  Backend API │  (Same server!)
└──────────────┘
```

---

## Key Points

### 1. CLI is Just a Client
- It's a standalone Go binary
- Runs on user's machine
- Makes HTTP calls to your API
- Like `curl` but with nice UI

### 2. Backend is the Brain
- Handles all business logic
- Manages rate limiting
- Sends blockchain transactions
- Serves both CLI and Frontend

### 3. One Backend, Multiple Clients
```
CLI (terminal) ──┐
                 ├──→ Backend API ──→ Starknet
Frontend (web) ──┘
```

### 4. Why npm for a Go Binary?
- npm is the most popular package manager
- Easy distribution: `npm install -g`
- Auto-updates
- Cross-platform
- Developers already have npm

### 5. Backend Environment
Deployed on Koyeb with these env vars:
```bash
STARKNET_RPC_URL=https://starknet-sepolia.g.alchemy.com/starknet/version/rpc/v0_9/...
FAUCET_PRIVATE_KEY=0x5f7c...
FAUCET_ADDRESS=0x31167a4...
REDIS_URL=redis://...upstash.io:6379
STRK_TOKEN_ADDRESS=0x04718f...
ETH_TOKEN_ADDRESS=0x049d36...
POW_DIFFICULTY=6
```

---

## Security Layers

1. **PoW Challenge** - CPU work prevents spam
2. **CAPTCHA** - Human verification
3. **IP Rate Limiting** - 5 requests/day per IP (Redis)
4. **Token Throttling** - 1 hour per token
5. **Challenge Expiry** - 5 minute TTL
6. **Private Key Security** - Stored in Koyeb secrets

---

## Questions?

Let me know if you want me to explain:
- How Redis stores rate limit data
- How the private key signing works
- How Alchemy routes to Starknet
- How to add new endpoints for your frontend
- Anything else!
