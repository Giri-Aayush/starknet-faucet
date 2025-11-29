# Architecture Overview

This doc explains how the faucet works end-to-end, from when someone runs `npm install` to when tokens hit their wallet.

## The Three Components

The faucet is split into three parts:

**CLI** - A Go binary that users install via npm. Runs on their machine.
**Backend** - API server on Koyeb. Validates requests and sends transactions.
**Frontend** - Your website that shows stats by calling the same backend API.

Think of it like this: the CLI and frontend are both clients talking to the same server.

---

## How Installation Works

When someone runs:
```bash
npm install -g starknet-faucet
```

Here's what happens:

1. npm downloads the package (5 KB - just has `install.js` and some metadata)
2. npm runs the postinstall script (`node install.js`)
3. `install.js` detects their OS and downloads the right binary:
   - Linux AMD64: 2.2 MB (UPX compressed)
   - Linux ARM64: 1.9 MB (UPX compressed)
   - macOS: 6.6 MB (can't compress on macOS)
   - Windows: 2.2 MB (UPX compressed)
4. Binary gets saved to `node_modules/starknet-faucet/bin/starknet-faucet`
5. npm creates a symlink in `/usr/local/bin` so it's in PATH

Now when they type `starknet-faucet`, the shell finds the binary and runs it.

The binary is just a compiled Go program - no runtime dependencies, no Node.js needed at runtime.

---

## Request Flow Walkthrough

Let me trace what happens when someone requests tokens. I'll use an actual example from testing:

```bash
starknet-faucet request 0x02ca67d3b01d9546a995880cc88173cd7335044f222370f047275b90c8e384fb
```

### Step 1: Local validation

The CLI parses the command and validates the address format. If it's malformed, it fails immediately without hitting the API.

```go
// pkg/cli/commands/request.go
if err := utils.ValidateStarknetAddress(address); err != nil {
    return fmt.Errorf("invalid address: %w", err)
}
```

### Step 2: CAPTCHA

The CLI shows a simple question like "What drink is made from coffee beans?"

This happens entirely locally - just reads from a hardcoded list of 100+ questions. The backend doesn't even know about it. It's just to slow down bots that might spam the CLI.

### Step 3: Get a challenge

Now the real flow starts. CLI makes an HTTP request:

```
GET https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app/api/v1/challenge
```

Backend generates a random hex string and stores it in Redis with a 5-minute expiry:

```go
// internal/api/handlers.go
challenge := generateRandomHex(32)
challengeID := uuid.New()
redis.Set("challenge:"+challengeID, challenge, 5*time.Minute)
```

Returns:
```json
{
  "challenge_id": "550e8400-e29b-41d4-a716-446655440000",
  "challenge": "0x4f8a2c9b3e1d7f6a...",
  "difficulty": 6
}
```

### Step 4: Solve proof of work (locally)

This is the interesting part. The CLI needs to find a number (nonce) where:

```
SHA256(challenge + nonce) starts with N zeros
```

With difficulty 6, that means the hash needs to start with "000000".

The CLI just brute forces it:

```go
// pkg/cli/pow/solver.go
for nonce := int64(0); ; nonce++ {
    hash := sha256.Sum256([]byte(challenge + strconv.FormatInt(nonce, 10)))
    hashHex := hex.EncodeToString(hash[:])

    if strings.HasPrefix(hashHex, strings.Repeat("0", difficulty)) {
        return nonce // Found it!
    }
}
```

On average this takes 20-60 seconds depending on your CPU. The user sees a spinner with progress updates.

The key point: **the user's CPU does this work, not the server**. That's the whole point of proof of work.

### Step 5: Submit the request

Now the CLI has:
- challenge_id
- nonce (the solution)
- address (where to send tokens)
- token (STRK, ETH, or BOTH)

It sends a POST request:

```
POST https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app/api/v1/request
Content-Type: application/json

{
  "address": "0x02ca67...",
  "token": "STRK",
  "challenge_id": "550e8400-e29b-41d4-a716-446655440000",
  "nonce": 800047
}
```

### Step 6: Backend validates everything

The backend does a bunch of checks (see `internal/api/handlers.go`):

**Check 1: IP rate limit**
```go
ip := c.IP()
canRequest, currentCount, cooldownEnd := redis.CheckIPDailyLimit(ip)
if !canRequest {
    return 429 // Too Many Requests
}
```

Someone can only make 5 requests per day from the same IP. After the 5th request, they're locked out for 24 hours.

**Check 2: Token throttle**
```go
canRequestToken, nextAvailable := redis.CheckTokenHourlyThrottle(ip, "STRK")
if !canRequestToken {
    return 429 // Too Many Requests
}
```

Even if you have quota left, you can only request the same token once per hour.

**Check 3: Verify the proof of work**
```go
storedChallenge := redis.Get("challenge:" + challengeID)
hash := sha256.Sum256([]byte(storedChallenge + strconv.FormatInt(nonce, 10)))
hashHex := hex.EncodeToString(hash[:])

if !strings.HasPrefix(hashHex, strings.Repeat("0", difficulty)) {
    return 400 // Bad Request - invalid PoW
}
```

If someone tries to skip the work and submit a random nonce, this check fails.

**Check 4: Delete the challenge**
```go
redis.Delete("challenge:" + challengeID)
```

Challenges are one-time use. Can't reuse the same solution.

If all checks pass, move to the next step.

### Step 7: Send the transaction

Now the backend needs to actually send tokens on Starknet. It uses the starknet.go library to build and sign a transaction:

```go
// internal/starknet/client.go
amount := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)) // 10 STRK

tx := account.BuildInvokeTransaction(
    tokenContract,
    "transfer",
    []interface{}{userAddress, amount},
)

signedTx := account.Sign(tx, privateKey) // Uses faucet private key from env

txHash, err := provider.AddInvokeTransaction(signedTx)
```

The `provider` is connected to Alchemy's Starknet RPC endpoint:
```
https://starknet-sepolia.g.alchemy.com/starknet/version/rpc/v0_9/cWsqpE1AYKEJM6-JaS28G
```

Alchemy forwards the transaction to the Starknet network. It goes into the mempool, gets picked up by a sequencer, and eventually gets included in a block (usually takes 10-30 seconds).

### Step 8: Return the response

Backend sends back:
```json
{
  "success": true,
  "tx_hash": "0x6139cd4b0c3fed17ea582d50453107b381153081b207b4cbad7e88f82e8d55f",
  "amount": "10",
  "token": "STRK",
  "explorer_url": "https://sepolia.voyager.online/tx/0x6139cd4b..."
}
```

CLI displays it nicely:
```
✓ Transaction submitted!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Amount:  10 STRK
  TX Hash:  0x6139cd4b...82e8d55f

  🔗 https://sepolia.voyager.online/tx/...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✓ Tokens will arrive in ~30 seconds.
```

Done. Tokens show up in the user's wallet about 30 seconds later once the transaction confirms.

---

## How the Frontend Fits In

The website just calls the same API endpoints:

```javascript
// Get faucet balance and info
fetch('https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app/api/v1/info')
  .then(r => r.json())
  .then(data => {
    // data.faucet_balance.strk
    // data.faucet_balance.eth
    // data.limits.daily_requests_per_ip
  });

// Check health
fetch('https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app/health')
  .then(r => r.json())
  .then(data => {
    // data.status === "ok"
  });
```

The backend has CORS enabled (`AllowOrigins: "*"`) so browsers can make these requests from any domain.

If you wanted to add a web-based faucet UI (not just stats), you'd use the same `/api/v1/challenge` and `/api/v1/request` endpoints. You'd just need to implement the PoW solver in JavaScript.

---

## API Endpoints Reference

| Endpoint | Method | Used By | Purpose |
|----------|--------|---------|---------|
| `/health` | GET | Frontend, CLI, monitoring | Health check |
| `/api/v1/info` | GET | Frontend | Get balance and limits |
| `/api/v1/challenge` | POST | CLI | Get PoW challenge |
| `/api/v1/request` | POST | CLI | Request tokens |
| `/api/v1/quota` | GET | CLI | Check user's quota |
| `/api/v1/limits` | GET | CLI | Get rate limit info |

Note: Some endpoints are POST that should probably be GET (like `/api/v1/challenge`), but that's just how it evolved. Might clean that up later.

---

## Deployment

The backend auto-deploys from GitHub to Koyeb:

1. Push to `main` branch
2. Koyeb detects the push (webhook)
3. Builds Docker image using `deployments/docker/Dockerfile.server`
4. Deploys to a container with these env vars:
   - `STARKNET_RPC_URL` - Alchemy endpoint
   - `FAUCET_PRIVATE_KEY` - Wallet that sends tokens
   - `REDIS_URL` - Upstash Redis (for rate limiting)
   - `STRK_TOKEN_ADDRESS` - STRK contract on Sepolia
   - `ETH_TOKEN_ADDRESS` - ETH contract on Sepolia
   - `POW_DIFFICULTY` - Currently set to 6

The whole deploy takes about 2-3 minutes. Old instances stay up until new ones pass health checks, so there's no downtime.

---

## Common Questions

**Q: Why distribute the CLI via npm if it's a Go binary?**
A: npm is the most popular package manager. Developers already have it. Alternative would be Homebrew (macOS only), apt (Linux only), or "download from GitHub" (annoying). npm works everywhere and handles global installs nicely.

**Q: Why is the API URL hardcoded in the CLI?**
A: Because the CLI runs on user's machines. It can't access your Koyeb environment variables. Users can override it with `--api-url` if they want to run their own instance.

**Q: Why solve PoW on the client instead of the server?**
A: DDoS protection. If the server had to do the work, an attacker could just spam requests and overload it. With client-side PoW, the attacker's CPU does the work. Each request costs them ~30 seconds of compute time.

**Q: Can't someone modify the CLI to skip PoW?**
A: They can modify their local copy, but the backend still validates the solution. If the hash doesn't start with 6 zeros, the request is rejected. You can't fake the math.

**Q: Why Redis instead of in-memory rate limiting?**
A: If the server restarts, in-memory state is lost. Redis persists rate limit data. Also, if we ever scale to multiple server instances, they can share the same Redis instance.

**Q: Why testnet only?**
A: Because the private key is in an environment variable, not a hardware wallet or MPC setup. That's fine for worthless testnet tokens, but you'd never do that with real money.

---

## Code Structure

```
cmd/
  cli/main.go           - CLI entry point
  server/main.go        - Backend entry point

pkg/cli/                - CLI-specific code
  commands/             - Cobra commands
  pow/solver.go         - PoW solver
  captcha/questions.go  - CAPTCHA questions
  ui/display.go         - Terminal UI

internal/               - Backend code
  api/
    handlers.go         - HTTP handlers
    routes.go           - Route setup
  starknet/client.go    - Starknet interaction
  cache/redis.go        - Redis client
  pow/pow.go            - PoW verification
  config/config.go      - Config loading

deployments/
  docker/Dockerfile.server  - Production Dockerfile
  docker-compose.yml        - Local dev setup
```

The `pkg/` vs `internal/` split is a Go convention:
- `pkg/` - Can be imported by other projects
- `internal/` - Private to this project

In practice, both are private here. Just following standard Go project layout.

---

That's the whole system. If you want to understand a specific part in more detail, check the code - it's pretty straightforward Go with minimal abstractions.
