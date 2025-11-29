# Koyeb Deployment Guide

## Prerequisites

1. **Koyeb Account**: Sign up at https://app.koyeb.com/
2. **GitHub Account**: Your repository must be on GitHub
3. **Upstash Redis**: Free Redis at https://upstash.com/ (Koyeb doesn't include Redis)

---

## Step 1: Set Up Upstash Redis

1. Go to https://console.upstash.com/
2. Click **"Create Database"**
3. Configuration:
   - Name: `starknet-faucet-redis`
   - Type: **Regional**
   - Region: **US-East-1** (same as Koyeb Washington region)
   - TLS: **Enabled**
4. Copy the **Redis URL** (looks like: `rediss://default:xxxxx@us1-xxxxx.upstash.io:6379`)

---

## Step 2: Deploy to Koyeb

### Option A: Deploy via Koyeb Dashboard (Recommended)

1. **Go to Koyeb**: https://app.koyeb.com/
2. **Click "Create App"**
3. **Select GitHub**:
   - Authorize Koyeb to access your GitHub
   - Select repository: `Giri-Aayush/starknet-faucet`
   - Branch: `main`
4. **Configure Build**:
   - Builder: **Dockerfile**
   - Dockerfile path: `deployments/docker/Dockerfile.server`
5. **Configure Service**:
   - Name: `faucet-api`
   - Port: `8080`
   - Health check: `/health`
6. **Add Environment Variables** (Secrets):

   Click "Add Variable" → "Secret" for each:

   ```
   STARKNET_RPC_URL
   Value: https://starknet-sepolia.g.alchemy.com/v2/cWsqpE1AYKEJM6-JaS28G

   FAUCET_PRIVATE_KEY
   Value: 0x359acd346217a48f1496af24e223abf7f476ed4d16f342a33b756c9a4a19fc6

   FAUCET_ADDRESS
   Value: 0x31167a418b40c646e0e77aff37d1af742472e07973e70e5043006a75763d430

   REDIS_URL
   Value: [Your Upstash Redis URL from Step 1]
   ```

   Add as **Plain Text**:
   ```
   PORT=8080
   NETWORK=sepolia
   LOG_LEVEL=info
   ETH_TOKEN_ADDRESS=0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7
   STRK_TOKEN_ADDRESS=0x04718f5a0Fc34cC1AF16A1cdee98fFB20C31f5cD61D6Ab07201858f4287c938D
   POW_DIFFICULTY=6
   CHALLENGE_TTL=300
   DRIP_AMOUNT_STRK=10
   DRIP_AMOUNT_ETH=0.01
   MAX_REQUESTS_PER_DAY_IP=5
   MAX_CHALLENGES_PER_HOUR=8
   MAX_TOKENS_PER_HOUR_STRK=500
   MAX_TOKENS_PER_DAY_STRK=10000
   MAX_TOKENS_PER_HOUR_ETH=5
   MAX_TOKENS_PER_DAY_ETH=100
   MIN_BALANCE_PROTECT_PCT=5
   ```

7. **Select Instance**:
   - Type: **Nano** (512MB RAM, 0.1 vCPU) - Free tier
   - Region: **Washington (was)**

8. **Click "Deploy"**

---

### Option B: Deploy via Koyeb CLI

```bash
# Install Koyeb CLI
curl -fsSL https://cli.koyeb.com/install.sh | bash

# Login
koyeb login

# Create secrets
koyeb secrets create STARKNET_RPC_URL --value "https://starknet-sepolia.g.alchemy.com/v2/cWsqpE1AYKEJM6-JaS28G"
koyeb secrets create FAUCET_PRIVATE_KEY --value "0x359acd346217a48f1496af24e223abf7f476ed4d16f342a33b756c9a4a19fc6"
koyeb secrets create FAUCET_ADDRESS --value "0x31167a418b40c646e0e77aff37d1af742472e07973e70e5043006a75763d430"
koyeb secrets create REDIS_URL --value "YOUR_UPSTASH_REDIS_URL"

# Deploy
koyeb app init starknet-faucet \
  --git github.com/Giri-Aayush/starknet-faucet \
  --git-branch main \
  --git-builder dockerfile \
  --git-dockerfile deployments/docker/Dockerfile.server \
  --ports 8080:http \
  --routes /:8080 \
  --env PORT=8080 \
  --env NETWORK=sepolia \
  --env LOG_LEVEL=info \
  --env ETH_TOKEN_ADDRESS=0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7 \
  --env STRK_TOKEN_ADDRESS=0x04718f5a0Fc34cC1AF16A1cdee98fFB20C31f5cD61D6Ab07201858f4287c938D \
  --env POW_DIFFICULTY=6 \
  --env CHALLENGE_TTL=300 \
  --env DRIP_AMOUNT_STRK=10 \
  --env DRIP_AMOUNT_ETH=0.01 \
  --env MAX_REQUESTS_PER_DAY_IP=5 \
  --env MAX_CHALLENGES_PER_HOUR=8 \
  --env MAX_TOKENS_PER_HOUR_STRK=500 \
  --env MAX_TOKENS_PER_DAY_STRK=10000 \
  --env MAX_TOKENS_PER_HOUR_ETH=5 \
  --env MAX_TOKENS_PER_DAY_ETH=100 \
  --env MIN_BALANCE_PROTECT_PCT=5 \
  --env-secret STARKNET_RPC_URL=@STARKNET_RPC_URL \
  --env-secret FAUCET_PRIVATE_KEY=@FAUCET_PRIVATE_KEY \
  --env-secret FAUCET_ADDRESS=@FAUCET_ADDRESS \
  --env-secret REDIS_URL=@REDIS_URL \
  --regions was \
  --instance-type nano
```

---

## Step 3: Get Your Koyeb URL

After deployment completes:

1. Go to Koyeb Dashboard
2. Click on your app: `starknet-faucet`
3. Copy the public URL (e.g., `https://starknet-faucet-yourapp.koyeb.app`)

---

## Step 4: Update Your npm Package

Update the default API URL in `pkg/cli/commands/root.go`:

```go
rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "https://YOUR-KOYEB-URL.koyeb.app", "Faucet API URL")
```

Then publish a new npm version.

---

## Step 5: Test Your Deployment

```bash
# Health check
curl https://YOUR-KOYEB-URL.koyeb.app/health

# Get faucet info
curl https://YOUR-KOYEB-URL.koyeb.app/api/v1/info

# Test with CLI
npm install -g starknet-faucet
starknet-faucet --api-url https://YOUR-KOYEB-URL.koyeb.app quota
```

---

## Monitoring & Logs

### View Logs
```bash
koyeb logs starknet-faucet/faucet-api
```

Or in the dashboard: **App** → **Logs**

### Monitor Metrics
Dashboard → **App** → **Metrics**
- CPU usage
- Memory usage
- Request count
- Response times

---

## Benefits of Koyeb vs Render

✅ **No cold starts** - Koyeb keeps apps warm
✅ **Better performance** - Faster response times
✅ **Free tier** - 512MB RAM, 2GB storage
✅ **Auto-deploy** - Pushes to main branch auto-deploy
✅ **Better logs** - Real-time log streaming
✅ **Health checks** - Automatic restarts on failure

---

## Costs

- **Free Tier**:
  - 1 Nano instance (512MB RAM)
  - 100GB bandwidth/month
  - Perfect for this faucet!

- **Upstash Redis**:
  - Free tier: 10,000 commands/day
  - Should be enough for faucet usage

---

## Troubleshooting

### Build Fails
- Check Dockerfile path is correct: `deployments/docker/Dockerfile.server`
- Ensure `go.mod` and `go.sum` are committed

### App Crashes
- Check logs: `koyeb logs starknet-faucet/faucet-api`
- Verify all environment variables are set
- Check Redis URL is correct

### Redis Connection Issues
- Ensure REDIS_URL includes `rediss://` (with TLS)
- Check Upstash database is active

### Alchemy RPC Issues
- If v0.9 endpoint still down, keep using v2
- Monitor Alchemy status page

---

## Migration from Render

1. ✅ Deploy to Koyeb (follow steps above)
2. ✅ Test thoroughly
3. ✅ Update DNS/documentation with new URL
4. ✅ Delete Render app (optional - or keep as backup)

---

## Auto-Deploy Setup

Koyeb automatically redeploys when you push to `main` branch:

1. Make changes locally
2. Commit: `git commit -m "Update"`
3. Push: `git push origin main`
4. Koyeb automatically builds and deploys!

---

## Support

- Koyeb Docs: https://www.koyeb.com/docs
- Koyeb Community: https://community.koyeb.com/
- Upstash Docs: https://docs.upstash.com/

---

**You're all set!** 🚀

Your faucet will be live at `https://starknet-faucet-XXXXX.koyeb.app` with:
- ✅ No timeouts
- ✅ No cold starts
- ✅ Auto-scaling
- ✅ Free hosting
