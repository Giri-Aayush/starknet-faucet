# Security Notes

Someone asked if having the API URL hardcoded in the CLI is a security risk. Short answer: no. Let me explain why.

## The URL Being Public is Fine

The API URL (`https://intermediate-albertine-aayushgiri-e93ace53.koyeb.app`) is in the CLI source code on GitHub. Anyone can see it. That's not a problem.

Think about it - your frontend also calls this API. It's right there in the browser dev tools. The URL being public is inherent to how HTTP APIs work.

Security through obscurity doesn't work. If your security relies on the URL being secret, you don't have security.

## How It Actually Works

The backend has multiple layers of protection. An attacker knowing the URL doesn't help them because:

### 1. IP Rate Limiting

Each IP can make 5 requests per day. After the 5th request, you're locked out for 24 hours.

```go
// internal/api/handlers.go:128
canRequest, currentCount, cooldownEnd := h.redis.CheckIPDailyLimit(ctx, ip)
if !canRequest && cooldownEnd != nil {
    return c.Status(429).JSON(...)
}
```

This is stored in Redis, so it persists across server restarts.

Could an attacker use a VPN to get new IPs? Sure, but:
- Each IP still limited to 5 requests
- That's 50 STRK (worth $0 on testnet)
- Costs them VPN fees or bot time
- Not economically viable

### 2. Token Throttling

Even if you have quota left, you can only request the same token once per hour.

```go
// internal/api/handlers.go:200
canRequestToken, nextAvailable := h.redis.CheckTokenHourlyThrottle(ctx, ip, req.Token)
if !canRequestToken {
    minutesRemaining := int(time.Until(*nextAvailable).Minutes())
    return c.Status(429).JSON(...)
}
```

So your 5 daily requests are spread out. Can't just spam 5 STRK requests in a row.

### 3. Proof of Work

This is the main anti-spam mechanism. Before you can request tokens, you need to:

1. Get a challenge from the server
2. Find a nonce where `SHA256(challenge + nonce)` starts with 6 zeros
3. Submit the nonce as proof you did the work

On average this takes 20-60 seconds of CPU time. The user's computer does this work, not the server.

```go
// internal/api/handlers.go:227
if !h.powGenerator.VerifyPoW(storedChallenge, req.Nonce, h.config.PoWDifficulty) {
    return c.Status(400).JSON(models.ErrorResponse{
        Error: "Invalid proof of work solution",
    })
}
```

Can someone skip this by modifying the CLI? Doesn't matter - the backend validates the solution. If you submit a random nonce, the math won't check out.

Could someone pre-solve a bunch of challenges? No - they expire after 5 minutes.

### 4. Challenge One-Time Use

After you use a challenge, it's deleted from Redis:

```go
// internal/api/handlers.go:239
if err := h.redis.DeleteChallenge(ctx, req.ChallengeID); err != nil {
    h.logger.Error("Failed to delete challenge", zap.Error(err))
}
```

So you can't reuse the same solution. Each request needs a fresh PoW solve.

### 5. Address and Token Validation

Basic input validation to prevent garbage data:

```go
// internal/api/handlers.go:109
if err := utils.ValidateStarknetAddress(req.Address); err != nil {
    return c.Status(400).JSON(...)
}

// internal/api/handlers.go:117
if err := utils.ValidateToken(req.Token); err != nil {
    return c.Status(400).JSON(...)
}
```

## What About CORS?

The backend has CORS set to allow all origins:

```go
// internal/api/routes.go:18
AllowOrigins: "*",
```

This is correct for a public API. The frontend needs to call it from browsers. The CLI calls it from users' machines (various IPs). Third-party tools might want to integrate.

If this was a private API with authentication, you'd restrict it. But for a public faucet, `*` is the right choice.

## Attack Scenarios

Let me walk through what happens if someone tries to abuse this.

### Scenario: Simple spam script

Attacker writes:
```bash
while true; do
  curl -X POST https://...koyeb.app/api/v1/request \
    -H "Content-Type: application/json" \
    -d '{"address":"0xATTACKER","token":"STRK"}'
done
```

What happens:
- No `challenge_id` provided → 400 Bad Request
- Even if they get a challenge, they need to solve PoW (~30s each)
- Rate limit: max 5 requests from their IP
- After 5th request: 24-hour lockout

Result: They get 50 STRK (worth $0 on testnet) and waste 2.5 minutes of CPU time.

### Scenario: Distributed attack

Attacker uses a botnet with 100 IPs. Each makes 5 requests.

What happens:
- 500 requests total
- Each requires ~30s of PoW solving
- Total CPU time: 4.2 hours across the botnet
- Drains: 5,000 STRK from faucet

Faucet currently has ~3,740 STRK, so after ~374 requests it's empty and stops working. That's annoying but not catastrophic - the server doesn't crash, it just runs out of tokens.

In practice, I'd notice this in the logs and could:
- Increase PoW difficulty (6 → 8 makes it ~4x harder)
- Ban specific addresses
- Add address-based rate limiting (in addition to IP)

But again - testnet tokens are worthless. Not a realistic attack scenario.

### Scenario: Replay attack

Attacker captures a valid request and tries to replay it:

```json
{
  "challenge_id": "abc123",
  "nonce": 800047,
  "address": "0x123...",
  "token": "STRK"
}
```

What happens:
- 1st attempt: ✓ Works
- 2nd attempt: ✗ Challenge deleted (400 error)
- 3rd+ attempts: ✗ Challenge not found

Challenges are one-time use, so replays don't work.

### Scenario: Man-in-the-middle

Attacker tries to intercept requests between CLI and backend.

What happens:
- Connection is HTTPS (encrypted)
- Browser/CLI validates SSL certificate
- Attacker can't read or modify traffic without breaking HTTPS
- Modern OS/browsers block invalid certs

MITM isn't possible without compromising the user's machine (at which point you have bigger problems).

## What's Actually Secret

These values are stored in Koyeb environment variables (not in code):

```bash
FAUCET_PRIVATE_KEY=0x5f7c...  # ← This would be really bad to leak
REDIS_URL=redis://...         # ← Contains password
STARKNET_RPC_URL=https://...  # ← Contains Alchemy API key
```

I've verified these aren't in the Git history:

```bash
$ git log --all --pretty=format: --name-only -- '.env*' | sort -u
.env.example  # ✓ Only the example file (no real secrets)

$ grep -r "0x5f7c" .
# No matches (good - means private key not in code)
```

The Alchemy API key being in the URL is a bit awkward (Alchemy's API design), but it's fine since:
- It's only in backend environment variables
- Backend doesn't expose it in API responses
- It's not in the CLI code or frontend

## What You Should NOT Do

**Don't add authentication to the faucet API**

I've seen faucets that require an API key or OAuth. This doesn't help:
- Key would be in CLI source (public on GitHub)
- Adds friction for users
- Doesn't prevent the attacks described above

**Don't obfuscate the API URL**

Some people try to base64-encode it or use string concatenation. This is pointless:
- Trivial to reverse
- Breaks transparency
- Doesn't prevent anything

**Don't store secrets in code**

Never do this:
```go
const PRIVATE_KEY = "0x..." // ← NEVER
```

You're already doing it correctly (environment variables), but worth stating explicitly.

## What You Should Do

**Monitor the faucet balance**

Add a check that alerts you when balance is low:

```go
balance, _ := h.starknet.GetBalance(ctx, "STRK")
if balance < 100 {
    h.logger.Warn("Faucet balance low", zap.String("balance", balance))
    // TODO: Send alert to Telegram/Discord
}
```

**Keep an eye on logs**

Koyeb gives you logs. Watch for:
- Same IP hitting rate limits repeatedly
- Failed PoW attempts (might indicate attack)
- Unusual traffic spikes

**Update dependencies periodically**

```bash
go get -u ./...
npm update
```

Security patches happen. Stay up to date.

## Potential Improvements

If you wanted to make this even more robust:

**Address-based rate limiting**

Currently only limiting by IP. Could also limit per address:

```go
addressRequests := h.redis.GetAddressRequestCount(req.Address, 24*time.Hour)
if addressRequests >= 2 {
    return c.Status(429).JSON(...)
}
```

Prevents someone from using the same address across multiple IPs.

**Dynamic PoW difficulty**

Adjust based on request rate:

```go
hourlyRate := h.redis.GetRequestRateLastHour()
if hourlyRate > 100 {
    h.config.PoWDifficulty = 8  // Harder
} else {
    h.config.PoWDifficulty = 6  // Normal
}
```

If someone's attacking, make it harder. If it's quiet, make it easier for legitimate users.

**Cloudflare in front**

Put Cloudflare (or similar) in front of Koyeb. Gets you:
- DDoS protection
- Bot detection
- Caching (for `/health` and `/info` endpoints)

But honestly for a testnet faucet, probably overkill.

## Summary

The API URL being public is not a security problem. The backend has multiple layers of protection:

1. IP rate limiting (5/day)
2. Token throttling (1/hour)
3. Proof of work (30s CPU per request)
4. Challenge expiry (5 min TTL)
5. One-time challenges
6. HTTPS encryption

An attacker can't bypass these by knowing the URL. They'd need to actually solve the PoW challenges and still be limited to 5 requests per IP per day.

For a testnet faucet distributing worthless tokens, this is more than sufficient.

If this was mainnet with real money, you'd want:
- Hardware wallet or MPC for the private key (not env var)
- More sophisticated rate limiting (maybe challenge-response systems)
- Monitoring and alerting
- Probably KYC or something (but then it's not really a faucet anymore)

But for Sepolia testnet, the current setup is fine.
