# Security Analysis - Starknet Faucet

## Question: "Can attackers exploit the hardcoded API URL?"

**Short Answer:** No. The URL being public is not a security vulnerability.

---

## Why Public URLs Are Safe

### 1. **Your API is MEANT to be public**
- Users need to access it from CLI
- Frontend needs to fetch data
- It's a faucet service, not a private API
- Hiding the URL provides no security benefit

### 2. **Security is in the backend, not in obscurity**

```
❌ Bad Security:  Hide the URL, hope no one finds it
✅ Good Security: Public URL + Rate limiting + PoW + Validation
```

---

## Your Defense Layers

### Layer 1: IP-Based Rate Limiting ⏱️

**Location:** `internal/api/handlers.go` line 125-160

**How it works:**
```
IP: 203.0.113.45
├─ Request 1: ✓ (1/5 used)
├─ Request 2: ✓ (2/5 used)
├─ Request 3: ✓ (3/5 used)
├─ Request 4: ✓ (4/5 used)
├─ Request 5: ✓ (5/5 used)
└─ Request 6: ✗ 24-hour cooldown activated
```

**Attack resistance:**
- Attacker can make max 5 requests per IP per day
- After 5th request: 24-hour lockout
- Stored in Redis (persists across server restarts)

**Cost to bypass:**
- Attacker needs new IP for every 5 requests
- VPNs/proxies cost money or get rate-limited too
- Not economical for worthless testnet tokens

---

### Layer 2: Per-Token Hourly Throttle ⏲️

**Location:** `internal/api/handlers.go` line 162-216

**How it works:**
```
IP requests STRK at 10:00 AM
├─ 10:00 AM: ✓ STRK sent
├─ 10:30 AM: ✗ Throttled (30 min remaining)
├─ 11:00 AM: ✓ STRK available again
└─ 11:05 AM: ✗ Throttled (55 min remaining)
```

**Why this matters:**
- Even if attacker has 5 requests/day quota
- Can only request same token once per hour
- Max 5 tokens per day (not 5 per hour)

---

### Layer 3: Proof of Work (PoW) 💪

**Location:** `internal/api/handlers.go` line 218-236

**How it works:**
```
1. User requests challenge from backend
   Backend: "Find nonce where SHA256(challenge+nonce) starts with 6 zeros"

2. User's computer works (20-60 seconds CPU time)
   Try nonce=0: hash=0x8a3f... ✗ (only 2 zeros)
   Try nonce=1: hash=0x5b2c... ✗ (only 3 zeros)
   ...
   Try nonce=800047: hash=0x000000a1... ✓ (6 zeros!)

3. User submits: {challenge_id, nonce: 800047}

4. Backend verifies:
   ✓ Challenge exists in Redis?
   ✓ SHA256(challenge+800047) starts with 6 zeros?
   ✓ Challenge not already used?
```

**Attack resistance:**
- Each request requires ~30 seconds of CPU work
- Attacker can't pre-compute (challenges expire in 5 min)
- Attacker can't skip (backend verifies solution)
- Attacker can't reuse (challenges deleted after use)

**DDoS protection:**
- Attacker's CPU does the work, not your server
- Rate limiting still applies (max 5 requests/day)
- Cost: 5 requests × 30s = 2.5 minutes CPU time for 50 STRK testnet tokens

---

### Layer 4: Challenge Expiry ⏳

**Location:** `internal/cache/redis.go`

**How it works:**
```
Time   Event
10:00  User gets challenge (expires at 10:05)
10:02  User solves PoW (nonce: 123456)
10:03  User submits request ✓
10:03  Challenge deleted from Redis (can't reuse)

OR

10:00  User gets challenge (expires at 10:05)
10:02  User solves PoW (nonce: 123456)
10:06  User submits request ✗ (challenge expired)
```

**Attack resistance:**
- Challenges expire after 5 minutes
- Can't pre-solve thousands of challenges
- Can't reuse solved challenges

---

### Layer 5: Address Validation ✅

**Location:** `internal/api/handlers.go` line 108-113

**How it works:**
```go
// Validate address format
if err := utils.ValidateStarknetAddress(req.Address); err != nil {
    return 400 Bad Request "Invalid address"
}
```

**Prevents:**
- Malformed addresses
- SQL injection attempts (if you had a DB)
- Random garbage data

---

### Layer 6: Token Validation ✅

**Location:** `internal/api/handlers.go` line 115-121

**How it works:**
```go
// Only allow: "STRK", "ETH", "BOTH"
if err := utils.ValidateToken(req.Token); err != nil {
    return 400 Bad Request "Invalid token"
}
```

**Prevents:**
- Requests for non-existent tokens
- Injection attacks via token field

---

### Layer 7: CORS Configuration 🌐

**Location:** `internal/api/routes.go` line 15-21

**Current config:**
```go
AllowOrigins: "*",  // Allow all domains
```

**Why this is OK:**
- Public API meant to be accessed from anywhere
- CLI (users' machines) need access
- Frontend (your website) needs access
- Third-party tools can integrate
- No sensitive data exposed

**When to restrict:**
- If you add admin endpoints
- If you add authentication
- If you expose private keys (NEVER do this!)

**Alternative (more restrictive):**
```go
// If you want to only allow specific domains
AllowOrigins: "https://yourwebsite.com, https://localhost:3000",
```

But this would break CLI usage (comes from random IPs/domains).

---

### Layer 8: Backend Environment Security 🔐

**What's protected:**
```bash
# These are SECRET (only on Koyeb server)
FAUCET_PRIVATE_KEY=0x5f7c...  # ← NEVER in Git!
REDIS_URL=redis://...         # ← NEVER in Git!
STARKNET_RPC_URL=https://...  # ← API key in URL

# These are PUBLIC (in CLI code)
API_URL=https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app  # ✓ OK to be public
```

**Verified safe:**
```bash
$ git log --all --pretty=format: --name-only -- '.env*' | sort -u
.env.example  # ✓ Only example file in Git

$ grep -r "FAUCET_PRIVATE_KEY" --exclude-dir=.git
# Only references to env variable, not actual key ✓
```

---

## Attack Scenarios & Outcomes

### Scenario 1: Script Kiddie Spam Attack

**Attack:**
```bash
# Attacker writes simple script
while true; do
  curl POST https://...koyeb.app/api/v1/request \
    -d '{"address":"0xATTACKER","token":"STRK"}'
done
```

**What happens:**
1. ❌ No challenge_id provided → 400 Bad Request
2. Even if they get challenge:
   - ❌ Must solve PoW (~30s each)
   - ❌ Rate limit: 5 requests max
   - ❌ 24-hour cooldown after 5th request
3. **Result:** Attacker gets 50 STRK (5 requests × 10 STRK) worth $0 (testnet)
4. **Cost to attacker:** 2.5 minutes CPU time + VPN costs

**Verdict:** Not economically viable ✓

---

### Scenario 2: Distributed Attack (Many IPs)

**Attack:**
```bash
# Attacker uses botnet with 100 IPs
# Each IP makes 5 requests = 500 total requests
```

**What happens:**
1. Each IP must solve PoW: 500 × 30s = 4.2 hours total CPU time
2. Drains faucet: 500 × 10 STRK = 5,000 STRK
3. Your faucet balance: 3,740 STRK
4. After ~374 requests, faucet is empty

**Your defenses:**
- Monitor logs for unusual patterns
- Set balance threshold alert
- Increase PoW difficulty if needed (difficulty 6 → 8)
- Ban specific addresses if needed
- Faucet runs out, stops working (not a server crash)

**Verdict:** Possible but expensive for attacker, low-value target ✓

---

### Scenario 3: Man-in-the-Middle (MITM)

**Attack:**
```
User → Attacker intercepts → Fake server
```

**Your defenses:**
- ✅ HTTPS encryption (SSL/TLS)
- ✅ Certificate validation
- ✅ User's browser/CLI verifies cert

**Outcome:**
- Attacker can't intercept without breaking HTTPS
- Modern browsers/OS block invalid certs

**Verdict:** Protected by HTTPS ✓

---

### Scenario 4: Replay Attack

**Attack:**
```bash
# Attacker captures valid request:
POST /api/v1/request
{
  "address": "0x123...",
  "challenge_id": "abc123",
  "nonce": 800047
}

# Tries to replay it 100 times
```

**Your defenses:**
```go
// Challenge deleted after first use
h.redis.DeleteChallenge(ctx, req.ChallengeID)
```

**Outcome:**
- 1st request: ✓ Success
- 2nd request: ✗ Challenge not found (deleted)
- 3rd+ requests: ✗ Challenge not found

**Verdict:** Protected by one-time challenges ✓

---

### Scenario 5: Frontend Domain Spoofing

**Attack:**
```
Attacker creates: fake-starknet-faucet.com
Attacker's site calls your API
```

**Your current CORS:**
```go
AllowOrigins: "*"  // Allows all domains
```

**What happens:**
- ✓ Attacker's site CAN call your API
- ✓ But still subject to all rate limits
- ✓ Can't bypass PoW
- ✓ Can't bypass IP limits
- ✓ User gets real tokens from your faucet

**Is this a problem?**
- No! Your API is meant to be public
- Attacker can't drain faucet (rate limits)
- Attacker can't steal keys (not exposed)
- It's like someone embedding Google Maps on their site

**Verdict:** Not a vulnerability ✓

---

## What You Should NOT Do

### ❌ DON'T: Add API Keys/Auth for Users

```go
// BAD IDEA:
if req.ApiKey != "secret123" {
    return 401 Unauthorized
}
```

**Why not:**
- API key would be in CLI code (public on GitHub)
- Attacker reads code, copies API key
- Doesn't prevent anything
- Adds complexity for users

### ❌ DON'T: Hide the URL

```go
// BAD IDEA:
const API_URL = obfuscate("https://...")
```

**Why not:**
- "Security through obscurity" is not security
- Attacker decompiles binary, finds URL
- Doesn't prevent any attacks
- Breaks legitimate integrations

### ❌ DON'T: Store Private Keys in Code

```go
// NEVER DO THIS:
const FAUCET_PRIVATE_KEY = "0x5f7c..."
```

**Why not:**
- Anyone can drain your entire wallet
- Attackers scan GitHub for private keys
- You're already doing this correctly (env vars) ✓

---

## What You SHOULD Do

### ✅ DO: Keep Private Keys in Environment Variables

```go
// CORRECT (what you're already doing):
privateKey := os.Getenv("FAUCET_PRIVATE_KEY")
```

**Stored in:**
- Koyeb Secrets (encrypted)
- Never in Git
- Never in code
- Never in logs

### ✅ DO: Monitor Unusual Activity

Check logs for:
- Same IP making many requests
- Same address requesting repeatedly
- Unusual traffic spikes
- Failed PoW attempts (could indicate attack)

### ✅ DO: Set Balance Alerts

```go
// Recommended: Add to handlers.go
if balance < minimumBalance {
    h.logger.Warn("Faucet balance low",
        zap.String("balance", balance))
    // Send alert to your email/Discord/Telegram
}
```

### ✅ DO: Keep Dependencies Updated

```bash
go get -u ./...  # Update Go dependencies
npm update       # Update npm dependencies
```

Security patches are released regularly.

---

## Security Checklist

- [x] Private keys stored in environment variables (not code)
- [x] HTTPS enabled (Koyeb provides SSL)
- [x] Rate limiting implemented (5/day per IP)
- [x] PoW challenge implemented (difficulty 6)
- [x] Challenge expiry implemented (5 min TTL)
- [x] Challenge one-time use (deleted after use)
- [x] IP-based throttling (1 hour per token)
- [x] Address validation
- [x] Token validation
- [x] CORS configured (allow all for public API)
- [x] No secrets in Git history
- [x] Logging enabled (Koyeb logs)
- [ ] Balance monitoring alerts (optional improvement)
- [ ] Automatic PoW difficulty adjustment (optional improvement)

---

## Recommended Improvements (Optional)

### 1. Add Balance Monitoring

```go
func (h *Handler) checkBalanceAndAlert(ctx context.Context) {
    balance, _ := h.starknet.GetBalance(ctx, "STRK")

    if balance < 100 {  // Less than 100 STRK remaining
        h.logger.Error("⚠️ FAUCET BALANCE LOW",
            zap.String("balance", balance))
        // TODO: Send email/Telegram alert
    }
}
```

### 2. Add Dynamic PoW Difficulty

```go
func (h *Handler) adjustPoWDifficulty() {
    requestRate := h.redis.GetRequestRateLastHour()

    if requestRate > 100 {  // Unusually high
        h.config.PoWDifficulty = 8  // Harder
    } else {
        h.config.PoWDifficulty = 6  // Normal
    }
}
```

### 3. Add Address-Based Rate Limiting

```go
// In addition to IP limiting
func (h *Handler) checkAddressLimit(address string) error {
    count := h.redis.GetAddressRequestCount(address, 24*time.Hour)

    if count > 2 {  // Max 2 requests per address per day
        return errors.New("address limit exceeded")
    }

    return nil
}
```

---

## Summary

### Is the hardcoded URL a security risk?

**No.** Here's why:

1. **Public APIs need public URLs** - CLI and frontend must access it
2. **Security is in the backend** - Rate limiting, PoW, validation
3. **Hiding URLs doesn't prevent attacks** - "Security through obscurity" is not security
4. **Your defenses work** - Multiple layers prevent abuse

### What actually matters for security:

✅ Private keys → Environment variables (not code)
✅ Rate limiting → 5/day per IP, 1/hour per token
✅ Proof of Work → CPU cost for attackers
✅ Challenge expiry → Can't pre-solve
✅ HTTPS → Encrypted communication

### What doesn't matter for security:

❌ Hiding the API URL
❌ Obfuscating the code
❌ Adding fake API keys

---

## Questions?

If you want to improve security further:
- Add balance monitoring alerts
- Add address-based rate limiting
- Add dynamic PoW difficulty adjustment
- Add IP reputation checking
- Add webhook for suspicious activity

But your current setup is **secure enough** for a testnet faucet. The value of testnet tokens is $0, so attackers have no economic incentive to bypass your defenses.
