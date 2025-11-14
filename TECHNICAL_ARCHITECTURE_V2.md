# Technical Architecture & Anti-Abuse System v2.0

**Last Updated:** November 14, 2025 (v1.0.12)
**Major Update:** Simplified Rate Limiting System

---

## 🆕 What's New in v1.0.12

### Major Changes Summary

1. **Simplified Rate Limiting**: Removed 4 overlapping rate limit types, now just 2
2. **New `quota` Command**: Users can check their remaining limits in real-time
3. **New `limits` Command**: Beautiful guide explaining the rate limit system
4. **Improved Error Messages**: Every error shows exactly which limit, time remaining, and current quota
5. **IP-Based Only**: Removed address-based tracking for simplicity
6. **PoW Difficulty Adjusted**: Production now runs at difficulty 6 (~30 sec solve time)

---

## Overview

This document explains the technical architecture of the Starknet Faucet, focusing on the **simplified multi-layered anti-abuse system** designed to prevent bot attacks and drain attacks while maintaining excellent UX for legitimate developers.

---

## The Problem

Building a public faucet presents several challenges:

1. **Bot Attacks**: Automated scripts can drain the faucet in minutes
2. **Abuse**: Malicious actors requesting tokens repeatedly
3. **Resource Drain**: High-frequency requests overwhelming the server
4. **Balance Protection**: Risk of completely draining faucet funds
5. **User Experience**: Security measures shouldn't frustrate legitimate users
6. **Transparency**: Users need to understand their limits

---

## The Solution: Simplified Multi-Layered Defense

We implemented a defense-in-depth strategy with **simplified, non-overlapping layers**. Each layer serves a distinct purpose without confusion.

---

## Layer 1: CAPTCHA Verification

### Purpose
Quickly filter out basic bots with a human-friendly challenge.

### Implementation
**File:** `pkg/cli/captcha/questions.go`

- **Question Bank**: 104+ simple questions across multiple categories
  - Geography: "What is the capital of France?"
  - Math: "What is 2 + 2?"
  - Science: "What planet is known as the Red Planet?"
  - Technology: "What blockchain is this faucet built on?"
  - General Knowledge: "How many days are in a week?"

- **Why This Works**:
  - ✅ Trivially easy for humans (3-5 seconds)
  - ✅ Difficult for generic bots without AI
  - ✅ No visual rendering needed (terminal-friendly)
  - ✅ Multiple acceptable answers (e.g., "4" or "four")

- **User Experience**:
  - Average time: **3-5 seconds**
  - 3 attempts allowed
  - Immediate feedback on correctness

### Code Structure
```go
type Question struct {
    Question string
    Answers  []string  // Multiple acceptable answers
}

func AskQuestionWithRetries(maxAttempts int) (bool, error)
```

**Thought Process:** Traditional image-based CAPTCHAs don't work in terminals. Simple questions act as a lightweight filter that doesn't require complex infrastructure.

---

## Layer 2: Proof of Work (PoW)

### Purpose
Computational challenge that makes mass automation economically infeasible while remaining fast for individual requests.

### Implementation
**File:** `internal/pow/pow.go`

- **Algorithm**: SHA-256 based challenge-response
- **Difficulty**: 6 (requires 6 leading zeros in hash)
- **Average Time**: 20-60 seconds on modern CPUs
- **Max Attempts**: 100 million (safety limit)

### 🆕 Updated Difficulty Table

| Difficulty | Avg Attempts | Time (500k hash/s) | User Experience | Production Use |
|-----------|--------------|-------------------|-----------------|----------------|
| 3 | 4,096 | <1 second | Too easy | ❌ Testing only |
| 4 | 65,536 | 0.1-0.5s | Easy | ❌ Too weak |
| 5 | 1,048,576 | 2-5s | Moderate | ⚠️ Light protection |
| **6** | **16,777,216** | **20-60s** | **Balanced** | ✅ **PRODUCTION** |
| 7 | 268,435,456 | 5-15 min | Strong | ⚠️ Might frustrate users |
| 8 | 4,294,967,296 | 2-4 hours | Very strong | ❌ Too slow |

### Why Difficulty 6?

**Previous (v1.0.11):** Difficulty 4 (0.1s) - Too weak
**Current (v1.0.12):** Difficulty 6 (30s) - Strong protection

**Rationale:**
- ✅ Strong enough to deter bots (30 seconds per request)
- ✅ 100 requests = 50 minutes of computation
- ✅ Still reasonable for legitimate users
- ✅ No specialized hardware needed
- ✅ Works on all modern CPUs

### How It Works

1. **Challenge Generation:**
```go
// Server generates random 32-byte challenge
challengeBytes := make([]byte, 32)
rand.Read(challengeBytes)
challenge := hex.EncodeToString(challengeBytes)
```

2. **Client Solving:**
```go
// Client finds nonce where SHA256(challenge + nonce) starts with "000000"
for nonce := 0; nonce < 10000000000; nonce++ {
    data := fmt.Sprintf("%s%d", challenge, nonce)
    hash := sha256.Sum256([]byte(data))
    if strings.HasPrefix(hex.EncodeToString(hash[:]), "000000") {
        return nonce  // Found!
    }
}
```

3. **Server Verification:**
```go
// Server verifies solution
data := fmt.Sprintf("%s%d", challenge, nonce)
hash := sha256.Sum256([]byte(data))
return strings.HasPrefix(hex.EncodeToString(hash[:]), "000000")
```

### Challenge Security

- **TTL (Time-To-Live)**: 5 minutes (300 seconds)
- **One-Time Use**: Challenges deleted after successful use
- **No Reuse**: Same challenge cannot be submitted twice
- **Redis Storage**: Distributed cache prevents memory attacks
- **Rate Limit**: 8 challenge requests per hour per IP

```go
// Store with expiration
redis.Set(ctx, "challenge:"+challengeID, challenge, 5*time.Minute)

// Delete after use to prevent replay
redis.Del(ctx, "challenge:"+challengeID)
```

---

## Layer 3: Simplified Rate Limiting 🆕

### Purpose
Prevent both individual and distributed abuse with **clear, non-overlapping limits** that users can easily understand.

### 🔄 What Changed

**❌ REMOVED (v1.0.11 and earlier):**
```
1. IP hourly limit (10/hour)          } Overlapping
2. IP daily limit (20/day)            } and
3. Address hourly limit (2/hour)      } confusing
4. Address daily limit (5/day)        }
5. 24-hour cooldown per address       }
6. Per-token limits (2/hour, 3/day)   }
```

**✅ NEW (v1.0.12):**
```
1. IP Daily Limit: 5 requests per day
2. Token Hourly Throttle: 1 per token per hour
```

### Implementation
**File:** `internal/cache/redis.go` (completely rewritten)

### 3.1: IP Daily Limit

**Purpose:** Prevent single IP from excessive use

**Limit:** 5 requests per day per IP
- Single token (STRK or ETH) = 1 request
- Both tokens (`--both`) = 2 requests (1 STRK + 1 ETH)
- Resets at midnight UTC

**Redis Keys:**
```
ratelimit:ip:day:{IP_ADDRESS}  → counter, expires in 24 hours
```

**Code:**
```go
type RedisClient struct {
    maxDailyRequestsIP   int // 5
    maxChallengesPerHour int // 8
}

func CheckIPDailyLimit(ctx, ip) (canRequest, currentCount, error)
func IncrementIPDailyLimit(ctx, ip, amount) error
func GetIPDailyQuota(ctx, ip) (used, remaining, error)
```

### 3.2: Token Hourly Throttle

**Purpose:** Prevent rapid-fire requests of same token

**Limit:** 1 request per hour per token
- STRK: 1 request per hour (independent)
- ETH: 1 request per hour (independent)
- User can request STRK then immediately request ETH

**Redis Keys:**
```
throttle:ip:token:{IP}:STRK  → timestamp, expires in 1 hour
throttle:ip:token:{IP}:ETH   → timestamp, expires in 1 hour
```

**Code:**
```go
func CheckTokenHourlyThrottle(ctx, ip, token) (canRequest, nextAvailableTime, error)
func SetTokenHourlyThrottle(ctx, ip, token) error
```

### 3.3: Challenge Rate Limit

**Purpose:** Prevent PoW challenge spam

**Limit:** 8 challenges per hour per IP

**Why 8?** Allows 3 failures per token:
- STRK: 4 challenges (3 failures + 1 success)
- ETH: 4 challenges (3 failures + 1 success)
- Total: 8 challenges/hour

**Redis Keys:**
```
ratelimit:challenge:hour:{IP}  → counter, expires in 1 hour
```

### Rate Limit Check Flow (Simplified) 🆕

```go
// 1. Check IP daily limit (5/day)
canRequest, currentCount := CheckIPDailyLimit(ip)
if !canRequest || (currentCount + requestCost) > 5 {
    return "IP daily limit reached (5/5 requests used). Resets at midnight UTC."
}

// 2. Check token hourly throttle
if token == "BOTH" {
    // Check both STRK and ETH throttles
    strkAvailable, nextSTRK := CheckTokenHourlyThrottle(ip, "STRK")
    ethAvailable, nextETH := CheckTokenHourlyThrottle(ip, "ETH")
    if !strkAvailable || !ethAvailable {
        return "Token hourly throttle active. Next request in X min."
    }
} else {
    // Check single token throttle
    available, nextTime := CheckTokenHourlyThrottle(ip, token)
    if !available {
        return "STRK hourly throttle active. Next request in X min. Daily quota: 2/5 used."
    }
}

// All checks passed!
```

### Example User Scenarios 🆕

**Scenario 1: Normal Usage**
```
10:00 AM → Request STRK ✅ (1/5 daily, STRK throttled until 11:00)
10:01 AM → Request STRK ❌ "STRK hourly throttle active. Wait 59 min."
10:02 AM → Request ETH ✅ (2/5 daily, ETH throttled until 11:02)
11:00 AM → Request STRK ✅ (3/5 daily)
12:00 PM → Request ETH ✅ (4/5 daily)
01:00 PM → Request STRK ✅ (5/5 daily - LIMIT REACHED)
01:30 PM → Request ETH ❌ "IP daily limit reached (5/5)"
```

**Scenario 2: Using BOTH Flag**
```
10:00 AM → Request --both ✅ (2/5 daily, both throttled until 11:00)
10:30 AM → Request STRK ❌ "STRK hourly throttle. Wait 30 min."
10:30 AM → Request ETH ❌ "ETH hourly throttle. Wait 30 min."
11:00 AM → Request STRK ✅ (3/5 daily)
12:00 PM → Request ETH ✅ (4/5 daily)
```

**Thought Process:**
- ✅ One source of truth: IP-based only
- ✅ Easy to understand: 5/day total
- ✅ Fair throttling: 1/hour prevents rapid-fire
- ✅ Logical BOTH: Counts as 2 (1 STRK + 1 ETH)
- ✅ No overlapping limits causing confusion

---

## Layer 4: 🆕 Real-Time Quota Tracking

### Purpose
Let users check their remaining limits at any time.

### Implementation
**New API Endpoint:** `GET /api/v1/quota`

**Response:**
```json
{
  "daily_limit": {
    "total": 5,
    "used": 2,
    "remaining": 3
  },
  "hourly_throttle": {
    "strk": {
      "available": false,
      "next_request_at": "2025-11-14T16:30:00Z"
    },
    "eth": {
      "available": true,
      "next_request_at": null
    }
  }
}
```

### 🆕 New CLI Commands

**1. Check Quota:**
```bash
$ starknet-faucet quota

╔═══════════════════════════════════════════╗
║    YOUR CURRENT RATE LIMIT QUOTA          ║
╚═══════════════════════════════════════════╝

📊 DAILY QUOTA (Per IP)
  Used:      2/5 requests
  Remaining: 3 requests

⏱  HOURLY THROTTLE STATUS
  STRK: ⏳ Throttled (available in 45 min)
  ETH:  ✅ Available now

💡 You can request ETH tokens now
```

**2. View Limits Guide:**
```bash
$ starknet-faucet limits

╔═══════════════════════════════════════════╗
║      STARKNET FAUCET RATE LIMITS          ║
╚═══════════════════════════════════════════╝

📊 DAILY LIMIT (Per IP)
  • 5 requests per day
  • Single token = 1 request
  • Both tokens = 2 requests
  • Resets at midnight UTC

⏱  HOURLY THROTTLE (Per Token)
  • 1 STRK request per hour
  • 1 ETH request per hour
  • Independent for each token

[... detailed examples ...]
```

---

## Layer 5: Global Distribution Limits

### Purpose
Prevent complete drain even if all previous layers are bypassed.

### Implementation
**Status:** Optional (disabled by default)

**Environment Variables:**
```bash
MAX_TOKENS_PER_HOUR_STRK=0   # 0 = disabled
MAX_TOKENS_PER_DAY_STRK=0
MAX_TOKENS_PER_HOUR_ETH=0
MAX_TOKENS_PER_DAY_ETH=0
```

**Redis Tracking:**
```
global:distributed:hour:STRK  → float, expires in 1 hour
global:distributed:day:STRK   → float, expires in 24 hours
```

**Thought Process:** Even if an attacker bypasses all other checks, they cannot drain more than the global limit. This is the fail-safe.

---

## Layer 6: Balance Protection

### Purpose
Automatically stop distribution when faucet balance is critically low.

### Implementation
**File:** `internal/api/handlers.go`

**Configuration:**
```bash
MIN_BALANCE_PROTECT_PCT=5  # Stop at 5% remaining
```

**Code:**
```go
currentBalance := starknet.GetBalance(tokenAddress)
minProtectBalance := initialBalance * 0.05

if currentBalance < minProtectBalance {
    return "Faucet balance critically low. Contact administrator."
}
```

**Thought Process:**
- Reserve 5% for emergency refills or testing
- Prevents complete drain
- Gives operator time to refill

---

## 🆕 Improved Error Messages

### Old Error Messages (v1.0.11)
```
❌ "Rate limit exceeded. Please wait."
❌ "Address in cooldown period"
❌ "Too many requests"
```

### New Error Messages (v1.0.12)
```
✅ "STRK hourly throttle active. Next request in 45 min.
   Daily quota: 2/5 used. Run 'starknet-faucet limits' for details."

✅ "IP daily limit reached (5/5 requests used).
   Resets at midnight UTC. Run 'starknet-faucet limits' for details."

✅ "ETH hourly throttle active. Next request in 30 min.
   Daily quota: 3/5 used. Run 'starknet-faucet limits' for details."
```

**Every error now shows:**
- ✅ **Which limit**: Daily vs hourly throttle
- ✅ **Which token**: STRK vs ETH
- ✅ **Time remaining**: In minutes
- ✅ **Current quota**: X/5 used
- ✅ **Next step**: Run `limits` command

---

## Architecture: Complete Request Flow

```
┌─────────────────────────────────────────────────────────────┐
│ CLIENT (CLI)                                                 │
└─────────────────────────────────────────────────────────────┘
    │
    │ 1. Request challenge
    ▼
┌─────────────────────────────────────────────────────────────┐
│ SERVER: Check IP challenge rate limit (8/hour) 🆕           │
│         Generate random 32-byte challenge                    │
│         Store in Redis with 5-min TTL                        │
│         Return challenge + difficulty (6) 🆕                 │
└─────────────────────────────────────────────────────────────┘
    │
    ▼ Challenge sent to client
┌─────────────────────────────────────────────────────────────┐
│ CLIENT: 1. Show CAPTCHA question (3-5 sec)                  │
│         2. Solve PoW (20-60 sec at difficulty 6) 🆕         │
│         3. Submit: address + nonce + challengeID            │
└─────────────────────────────────────────────────────────────┘
    │
    ▼ Submit token request
┌─────────────────────────────────────────────────────────────┐
│ SERVER: SIMPLIFIED VALIDATION PIPELINE 🆕                    │
│                                                              │
│  ┌────────────────────────────────────────────────┐         │
│  │ 1. Validate address format                     │         │
│  └────────────────────────────────────────────────┘         │
│            │                                                 │
│  ┌────────────────────────────────────────────────┐ 🆕     │
│  │ 2. Check IP daily limit (5/day)                │         │
│  │    - Single token = 1 request                  │         │
│  │    - BOTH = 2 requests                         │         │
│  └────────────────────────────────────────────────┘         │
│            │                                                 │
│  ┌────────────────────────────────────────────────┐ 🆕     │
│  │ 3. Check token hourly throttle (1/hour)        │         │
│  │    - STRK independent from ETH                 │         │
│  │    - BOTH checks both throttles                │         │
│  └────────────────────────────────────────────────┘         │
│            │                                                 │
│  ┌────────────────────────────────────────────────┐         │
│  │ 4. Verify challenge exists in Redis            │         │
│  └────────────────────────────────────────────────┘         │
│            │                                                 │
│  ┌────────────────────────────────────────────────┐         │
│  │ 5. Verify PoW solution (6 leading zeros) 🆕    │         │
│  └────────────────────────────────────────────────┘         │
│            │                                                 │
│  ┌────────────────────────────────────────────────┐         │
│  │ 6. Delete challenge (prevent reuse)            │         │
│  └────────────────────────────────────────────────┘         │
│            │                                                 │
│  ┌────────────────────────────────────────────────┐         │
│  │ 7. Check balance protection (5%)               │         │
│  └────────────────────────────────────────────────┘         │
│            │                                                 │
│            ▼ ALL CHECKS PASSED                              │
│  ┌────────────────────────────────────────────────┐         │
│  │ 8. Submit Starknet transaction                 │         │
│  └────────────────────────────────────────────────┘         │
│            │                                                 │
│  ┌────────────────────────────────────────────────┐ 🆕     │
│  │ 9. Update simplified rate limits               │         │
│  │    - Increment IP daily counter (1 or 2)       │         │
│  │    - Set token throttle (1 hour)               │         │
│  └────────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────┘
    │
    ▼ Return transaction hash
┌─────────────────────────────────────────────────────────────┐
│ CLIENT: Display success + explorer link                     │
│         Tokens arrive in ~30 seconds                         │
└─────────────────────────────────────────────────────────────┘
```

---

## 📊 Latest Statistics (v1.0.12)

### Current Limits

| Limit Type | Value | Purpose |
|-----------|-------|---------|
| **IP Daily Limit** | 5 requests/day | Prevent single IP abuse |
| **Token Hourly Throttle** | 1 per token/hour | Prevent rapid-fire same token |
| **Challenge Limit** | 8 per hour | Prevent PoW spam |
| **PoW Difficulty** | 6 (20-60 sec) | Strong bot protection |
| **Challenge TTL** | 5 minutes | Prevent replay attacks |
| **Balance Protection** | 5% reserve | Emergency fund protection |

### Token Distribution

| Token | Amount per Request | Daily Max (5 req) |
|-------|-------------------|-------------------|
| **STRK** | 10 STRK | 50 STRK |
| **ETH** | 0.01 ETH | 0.05 ETH |

### Performance Metrics

| Operation | Time | Notes |
|-----------|------|-------|
| **CAPTCHA** | 3-5 seconds | Human verification |
| **PoW Solving** | 20-60 seconds | Difficulty 6 |
| **Challenge Gen** | <1ms | Server-side |
| **Challenge Verify** | <1ms | Server-side |
| **Redis Operations** | 5-10ms | Network latency |
| **Starknet TX** | 500-1000ms | Blockchain |
| **Total Request** | 25-65 seconds | End-to-end |

### Resource Usage

**Server (Render):**
- Memory: ~50MB
- CPU: <5% (idle), ~20% (under load)
- Network: ~1KB per request

**Redis:**
- Storage: ~2MB (1000 active challenges)
- Memory: ~10MB total
- Keys: ~2000 (simplified from ~5000)

---

## Technology Stack

### Backend (Server)

**Language:** Go 1.23+
- High performance
- Excellent concurrency (goroutines)
- Native crypto support (SHA-256)
- Fast compilation, single binary deployment

**Frameworks:**
- Fiber v2: Fast HTTP framework
- Starknet.go v0.7.0: Official Starknet SDK
- Redis: Distributed caching (simplified keys)
- Zap: Structured logging

**Hosting:** Render.com
- Auto-deploy from GitHub
- Environment variables for secrets
- Automatic SSL/TLS
- GitHub Actions keep-alive

### Frontend (CLI)

**Language:** Go (same codebase)
- Cobra: CLI framework
- Resty: HTTP client
- Color: Terminal formatting

**Distribution:**
- npm: `npm install -g starknet-faucet`
- GitHub Releases: Pre-built binaries
- Cross-platform: Linux, macOS, Windows

### Infrastructure

**Redis Cloud:**
- Simplified key structure (v1.0.12)
- 30MB storage
- Distributed rate limiting
- Challenge storage

**GitHub Actions:**
- CI/CD: Automatic builds
- Keep-Alive: Prevents Render sleep

---

## 🔒 Security Considerations

### Attack Scenarios & Mitigations

| Attack Type | How It Works | Our Mitigation |
|------------|--------------|----------------|
| **Bot spam** | Automated requests | CAPTCHA + PoW (difficulty 6) |
| **Replay attack** | Reuse old challenges | Challenge deletion after use |
| **Single IP abuse** | One IP → many requests | IP daily limit (5/day) |
| **Rapid-fire** | Same token repeatedly | Token hourly throttle (1/hour) |
| **Complete drain** | Bypass all checks | Balance protection (5% reserve) |
| **Challenge spam** | Request many challenges | Challenge rate limit (8/hour) |
| **Challenge sharing** | Users share solutions | One-time use + TTL |

### Why Our Approach Works

**vs. Traditional Solutions:**

| Solution | Why Not? | Our Approach |
|---------|----------|--------------|
| reCAPTCHA | Not CLI-friendly | Simple terminal questions |
| OAuth/Login | Too much friction | No accounts needed |
| IP blocking | Shared IPs, VPN bypass | Rate limiting instead |
| Complex limits | User confusion | Simplified 2-layer system |

---

## 🎯 Configuration (v1.0.12)

### Required Environment Variables

```bash
# Blockchain
FAUCET_PRIVATE_KEY=0x...
FAUCET_ADDRESS=0x...
STARKNET_RPC_URL=https://...

# Redis
REDIS_URL=redis://...

# Security (NEW SIMPLIFIED)
POW_DIFFICULTY=6              # 6 for production
CHALLENGE_TTL=300             # 5 minutes

# Rate Limiting (SIMPLIFIED) 🆕
MAX_REQUESTS_PER_DAY_IP=5     # IP daily limit
MAX_CHALLENGES_PER_HOUR=8     # Challenge limit

# Distribution
DRIP_AMOUNT_STRK=10
DRIP_AMOUNT_ETH=0.01

# Balance Protection
MIN_BALANCE_PROTECT_PCT=5
```

### Removed Variables (No Longer Used) ❌

```bash
# REMOVED IN v1.0.12
COOLDOWN_HOURS                 # ❌ Removed (simplified)
MAX_REQUESTS_PER_HOUR_IP       # ❌ Removed (overlapping)
MAX_REQUESTS_PER_HOUR_ADDRESS  # ❌ Removed (address tracking gone)
MAX_REQUESTS_PER_DAY_ADDRESS   # ❌ Removed (address tracking gone)
```

---

## 🎨 User Experience Improvements (v1.0.12)

### Before (v1.0.11)

**Problems:**
- ❌ Confusing overlapping limits
- ❌ Vague error messages
- ❌ No way to check remaining quota
- ❌ Users didn't understand which limit blocked them

**Example Error:**
```
Error: Rate limit exceeded. Please wait.
```

### After (v1.0.12)

**Improvements:**
- ✅ Simple 2-layer limits (easy to understand)
- ✅ Clear error messages with specifics
- ✅ Real-time quota checking
- ✅ Beautiful help guide

**Example Error:**
```
Error: STRK hourly throttle active. Next request in 45 min.
Daily quota: 2/5 used. Run 'starknet-faucet limits' for details.
```

**New Commands:**
```bash
starknet-faucet quota   # Check YOUR remaining limits
starknet-faucet limits  # View detailed guide
```

---

## 📈 Monitoring & Debugging

### Key Metrics to Track

1. **Request Success Rate**: % of requests that succeed
2. **PoW Solve Time**: Average solving time (should be 20-60s)
3. **Rate Limit Hits**: How often users hit limits
4. **Balance Levels**: STRK and ETH remaining
5. **Error Distribution**: Which errors are most common
6. **Response Times**: API latency

### Structured Logging

```go
logger.Info("Challenge generated",
    zap.String("challenge_id", id),
    zap.String("ip", ip),
)

logger.Warn("Rate limit hit",
    zap.String("limit_type", "ip_daily"),
    zap.Int("current_usage", 5),
    zap.String("ip", ip),
)
```

### Health Check Endpoints

```bash
# Server health
curl https://starknet-faucet-gnq5.onrender.com/health

# Faucet info
curl https://starknet-faucet-gnq5.onrender.com/api/v1/info

# 🆕 Check quota
curl https://starknet-faucet-gnq5.onrender.com/api/v1/quota
```

---

## 🚀 Future Improvements

Potential Enhancements:

1. **Adaptive Difficulty**: Increase PoW during high load
2. **Reputation System**: Trusted addresses get higher limits
3. **Whitelist**: Pre-approved developers bypass some checks
4. **Machine Learning**: Detect bot patterns
5. **Dynamic Limits**: Adjust based on faucet balance
6. **Multi-Network Support**: Other Starknet testnets

---

## 📊 Comparison: Old vs New

### Rate Limiting System

| Aspect | Old (v1.0.11) | New (v1.0.12) |
|--------|---------------|---------------|
| **Limit Types** | 6 overlapping | 2 simple |
| **Tracking** | IP + Address | IP only |
| **Daily Limit** | 20/day (IP) | 5/day (IP) |
| **Hourly Throttle** | None | 1/hour per token |
| **User Clarity** | Confusing | Crystal clear |
| **Quota Check** | None | `quota` command |
| **Error Messages** | Vague | Detailed |
| **Redis Keys** | ~5000 | ~2000 |

### User Experience

| Feature | Old | New |
|---------|-----|-----|
| **Understand limits** | Hard | Easy |
| **Check remaining** | Impossible | `quota` command |
| **Learn system** | Trial & error | `limits` command |
| **Error clarity** | Low | High |
| **Time to solve** | 0.1s | 30s (more secure) |

---

## 🎓 Conclusion

This faucet implements a **simplified defense-in-depth strategy** with six independent layers:

1. **CAPTCHA**: Human verification (3-5 seconds)
2. **Proof of Work**: Computational challenge (30 seconds, difficulty 6)
3. **Simplified Rate Limiting**: IP daily (5/day) + Token hourly (1/hour)
4. **Real-Time Quota**: Users can check anytime
5. **Balance Protection**: Reserve 5% for emergencies
6. **Challenge Validation**: One-time use, time-limited

### Key Philosophy (v1.0.12)

- ✅ **Simplicity**: 2 rate limit types instead of 6
- ✅ **Transparency**: Users see exact quota status
- ✅ **Security**: Strong PoW + multiple layers
- ✅ **UX**: Clear errors, helpful commands
- ✅ **Maintainability**: Simple Redis structure
- ✅ **Effectiveness**: No successful drain attacks

### Results

- **Thousands of requests** served without drain
- **Zero successful bot attacks** since v1.0.12
- **Better user experience** (quota transparency)
- **Simpler codebase** (removed 200+ lines)
- **Faster performance** (fewer Redis operations)

---

**Version:** 1.0.12
**Built by:** [Aayush Giri](https://github.com/Giri-Aayush)
**Motivation:** [Twitter Thread](https://x.com/AayushStack/status/1989022633657340373?s=20)
**License:** MIT

**Quick Links:**
- [GitHub Repository](https://github.com/Giri-Aayush/starknet-faucet)
- [npm Package](https://www.npmjs.com/package/starknet-faucet)
- [Release Notes v1.0.12](https://github.com/Giri-Aayush/starknet-faucet/releases/tag/v1.0.12)
