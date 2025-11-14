package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/Giri-Aayush/starknet-faucet/internal/cache"
	"github.com/Giri-Aayush/starknet-faucet/internal/config"
	"github.com/Giri-Aayush/starknet-faucet/internal/models"
	"github.com/Giri-Aayush/starknet-faucet/internal/pow"
	"github.com/Giri-Aayush/starknet-faucet/internal/starknet"
	"github.com/Giri-Aayush/starknet-faucet/pkg/utils"
	"go.uber.org/zap"
)

// Handler contains dependencies for API handlers
type Handler struct {
	config        *config.Config
	logger        *zap.Logger
	redis         *cache.RedisClient
	starknet      *starknet.FaucetClient
	powGenerator  *pow.Generator
}

// NewHandler creates a new API handler
func NewHandler(
	cfg *config.Config,
	logger *zap.Logger,
	redis *cache.RedisClient,
	starknetClient *starknet.FaucetClient,
	powGenerator *pow.Generator,
) *Handler {
	return &Handler{
		config:       cfg,
		logger:       logger,
		redis:        redis,
		starknet:     starknetClient,
		powGenerator: powGenerator,
	}
}

// GetChallenge generates a new PoW challenge
func (h *Handler) GetChallenge(c *fiber.Ctx) error {
	ctx := context.Background()

	// Check challenge rate limit for this IP
	ip := c.IP()
	canRequest, err := h.redis.CheckChallengeRateLimit(ctx, ip)
	if err != nil {
		h.logger.Error("Failed to check challenge rate limit", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to check rate limit",
		})
	}
	if !canRequest {
		return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
			Error: "Too many challenge requests. Please try again later.",
		})
	}

	// Generate challenge
	response, challenge, err := h.powGenerator.GenerateChallenge()
	if err != nil {
		h.logger.Error("Failed to generate challenge", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to generate challenge",
		})
	}

	// Store challenge in Redis
	ttl := time.Duration(h.config.ChallengeTTL) * time.Second
	if err := h.redis.StoreChallenge(ctx, challenge.ID, challenge.Challenge, ttl); err != nil {
		h.logger.Error("Failed to store challenge", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to store challenge",
		})
	}

	// Increment challenge rate limit counter
	if err := h.redis.IncrementChallengeRateLimit(ctx, ip); err != nil {
		h.logger.Error("Failed to increment challenge rate limit", zap.Error(err))
	}

	h.logger.Info("Challenge generated",
		zap.String("challenge_id", challenge.ID),
		zap.String("ip", ip),
	)

	return c.JSON(response)
}

// RequestTokens handles faucet requests
func (h *Handler) RequestTokens(c *fiber.Ctx) error {
	ctx := context.Background()

	// Parse request
	var req models.FaucetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "Invalid request body",
		})
	}

	// Validate address
	if err := utils.ValidateStarknetAddress(req.Address); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: fmt.Sprintf("Invalid address: %s", err.Error()),
		})
	}

	// Validate token
	req.Token = strings.ToUpper(req.Token)
	if err := utils.ValidateToken(req.Token); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: err.Error(),
		})
	}

	// NEW SIMPLIFIED RATE LIMITING

	ip := c.IP()

	// 1. Check IP daily limit (5 requests/day) and 24h cooldown
	canRequest, currentCount, cooldownEnd, err := h.redis.CheckIPDailyLimit(ctx, ip)
	if err != nil {
		h.logger.Error("Failed to check IP daily limit", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to check rate limit",
		})
	}

	// If in 24h cooldown after hitting limit
	if !canRequest && cooldownEnd != nil {
		hoursRemaining := time.Until(*cooldownEnd).Hours()
		errorMsg := fmt.Sprintf("Daily limit reached. In 24-hour cooldown (%.1f hours remaining). Run 'starknet-faucet limits' for details.",
			hoursRemaining)
		return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
			Error: errorMsg,
		})
	}

	// Calculate how many requests this will consume (1 for single token, 2 for BOTH)
	requestCost := 1
	if req.Token == "BOTH" {
		requestCost = 2
	}

	// Check if there's enough quota
	if !canRequest || (currentCount+requestCost) > h.config.MaxRequestsPerDayIP {
		used, _, _, _ := h.redis.GetIPDailyQuota(ctx, ip)
		errorMsg := fmt.Sprintf("IP daily limit reached (%d/%d requests used). Run 'starknet-faucet limits' for details.",
			used, h.config.MaxRequestsPerDayIP)
		return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
			Error: errorMsg,
		})
	}

	// 2. Check per-token hourly throttle
	if req.Token == "BOTH" {
		// For BOTH, check both STRK and ETH throttles
		canRequestSTRK, nextSTRK, err := h.redis.CheckTokenHourlyThrottle(ctx, ip, "STRK")
		if err != nil {
			h.logger.Error("Failed to check STRK throttle", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
				Error: "Failed to check rate limit",
			})
		}
		if !canRequestSTRK {
			minutesRemaining := int(time.Until(*nextSTRK).Minutes())
			used, _, _, _ := h.redis.GetIPDailyQuota(ctx, ip)
			errorMsg := fmt.Sprintf("STRK hourly throttle active. Next request in %d min. Daily quota: %d/%d used. Run 'starknet-faucet limits' for details.",
				minutesRemaining, used, h.config.MaxRequestsPerDayIP)
			return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
				Error: errorMsg,
			})
		}

		canRequestETH, nextETH, err := h.redis.CheckTokenHourlyThrottle(ctx, ip, "ETH")
		if err != nil {
			h.logger.Error("Failed to check ETH throttle", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
				Error: "Failed to check rate limit",
			})
		}
		if !canRequestETH {
			minutesRemaining := int(time.Until(*nextETH).Minutes())
			used, _, _, _ := h.redis.GetIPDailyQuota(ctx, ip)
			errorMsg := fmt.Sprintf("ETH hourly throttle active. Next request in %d min. Daily quota: %d/%d used. Run 'starknet-faucet limits' for details.",
				minutesRemaining, used, h.config.MaxRequestsPerDayIP)
			return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
				Error: errorMsg,
			})
		}
	} else {
		// For single token, check that token's throttle
		canRequestToken, nextAvailable, err := h.redis.CheckTokenHourlyThrottle(ctx, ip, req.Token)
		if err != nil {
			h.logger.Error("Failed to check token throttle", zap.Error(err), zap.String("token", req.Token))
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
				Error: "Failed to check rate limit",
			})
		}
		if !canRequestToken {
			minutesRemaining := int(time.Until(*nextAvailable).Minutes())
			used, _, _, _ := h.redis.GetIPDailyQuota(ctx, ip)
			errorMsg := fmt.Sprintf("%s hourly throttle active. Next request in %d min. Daily quota: %d/%d used. Run 'starknet-faucet limits' for details.",
				req.Token, minutesRemaining, used, h.config.MaxRequestsPerDayIP)
			return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
				Error: errorMsg,
			})
		}
	}

	// Verify challenge exists
	storedChallenge, err := h.redis.GetChallenge(ctx, req.ChallengeID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "Invalid or expired challenge",
		})
	}

	// Verify PoW solution
	if !h.powGenerator.VerifyPoW(storedChallenge, req.Nonce, h.config.PoWDifficulty) {
		h.logger.Warn("Invalid PoW solution",
			zap.String("challenge_id", req.ChallengeID),
			zap.Int64("nonce", req.Nonce),
			zap.String("ip", ip),
		)
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "Invalid proof of work solution",
		})
	}

	// Delete challenge to prevent reuse
	if err := h.redis.DeleteChallenge(ctx, req.ChallengeID); err != nil {
		h.logger.Error("Failed to delete challenge", zap.Error(err))
	}

	// Handle BOTH token request
	if req.Token == "BOTH" {
		return h.handleBothTokensRequest(c, ctx, req, ip)
	}

	// Determine amount (single token)
	var amountStr string
	var amountFloat float64
	var maxHourly, maxDaily float64
	if req.Token == "STRK" {
		amountStr = h.config.DripAmountSTRK
		amountFloat, _ = strconv.ParseFloat(amountStr, 64)
		maxHourly = h.config.MaxTokensPerHourSTRK
		maxDaily = h.config.MaxTokensPerDaySTRK
	} else {
		amountStr = h.config.DripAmountETH
		amountFloat, _ = strconv.ParseFloat(amountStr, 64)
		maxHourly = h.config.MaxTokensPerHourETH
		maxDaily = h.config.MaxTokensPerDayETH
	}

	// Check global distribution limits (anti-drain protection)
	canDistribute, err := h.redis.TrackGlobalDistribution(ctx, req.Token, amountFloat, maxHourly, maxDaily)
	if err != nil {
		h.logger.Error("Failed to check global distribution limits", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to process request",
		})
	}
	if !canDistribute {
		h.logger.Warn("Global distribution limit reached",
			zap.String("token", req.Token),
			zap.String("ip", ip),
		)
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.ErrorResponse{
			Error: "Faucet has reached its distribution limit. Please try again later.",
		})
	}

	// Check minimum balance protection (stop at configured percentage)
	currentBalance, err := h.starknet.GetBalance(ctx, h.config.FaucetAddress, req.Token)
	if err != nil {
		h.logger.Error("Failed to check faucet balance", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to check faucet balance",
		})
	}

	// Convert amount to wei for comparison
	amountWei := starknet.AmountToWei(amountFloat)

	// Check if balance would drop below minimum threshold
	minBalancePct := float64(h.config.MinBalanceProtectPct) / 100.0
	currentBalanceFloat := starknet.WeiToAmount(currentBalance)
	minBalanceRequired := currentBalanceFloat * minBalancePct
	balanceAfterTransfer := currentBalanceFloat - amountFloat

	if balanceAfterTransfer < minBalanceRequired {
		h.logger.Warn("Balance protection triggered",
			zap.String("token", req.Token),
			zap.Float64("current_balance", currentBalanceFloat),
			zap.Float64("min_balance_required", minBalanceRequired),
			zap.String("ip", ip),
		)
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.ErrorResponse{
			Error: fmt.Sprintf("Faucet balance too low. Current %s balance: %.4f", req.Token, currentBalanceFloat),
		})
	}

	// Transfer tokens
	h.logger.Info("Transferring tokens",
		zap.String("recipient", req.Address),
		zap.String("token", req.Token),
		zap.String("amount", amountStr),
		zap.String("ip", ip),
	)

	txHash, err := h.starknet.TransferTokens(ctx, req.Address, req.Token, amountWei)
	if err != nil {
		h.logger.Error("Failed to transfer tokens",
			zap.Error(err),
			zap.String("recipient", req.Address),
			zap.String("token", req.Token),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to send tokens. Please try again later.",
		})
	}

	// Increment IP daily counter (1 for single token)
	if err := h.redis.IncrementIPDailyLimit(ctx, ip, 1); err != nil {
		h.logger.Error("Failed to increment IP daily limit", zap.Error(err))
	}

	// Set token hourly throttle (1 hour cooldown for this token)
	if err := h.redis.SetTokenHourlyThrottle(ctx, ip, req.Token); err != nil {
		h.logger.Error("Failed to set token throttle", zap.Error(err))
	}

	// Build response
	response := models.FaucetResponse{
		Success:     true,
		TxHash:      txHash,
		Amount:      amountStr,
		Token:       req.Token,
		ExplorerURL: h.config.GetExplorerURL(txHash),
		Message:     "Tokens sent successfully",
	}

	h.logger.Info("Tokens sent successfully",
		zap.String("tx_hash", txHash),
		zap.String("recipient", req.Address),
		zap.String("token", req.Token),
	)

	return c.JSON(response)
}

// GetStatus returns the status of an address
func (h *Handler) GetStatus(c *fiber.Ctx) error {
	ctx := context.Background()

	address := c.Params("address")

	// Validate address
	if err := utils.ValidateStarknetAddress(address); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: fmt.Sprintf("Invalid address: %s", err.Error()),
		})
	}

	// Get IP from request (status endpoint doesn't have strict auth, just returns info)
	ip := c.IP()

	// Get IP daily quota
	used, remaining, cooldownEnd, err := h.redis.GetIPDailyQuota(ctx, ip)
	if err != nil {
		h.logger.Error("Failed to get IP daily quota", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to check status",
		})
	}

	canRequest := remaining > 0 && cooldownEnd == nil

	response := models.StatusResponse{
		Address:    address,
		CanRequest: canRequest,
	}

	h.logger.Info("Status check",
		zap.String("address", address),
		zap.String("ip", ip),
		zap.Int("daily_quota_used", used),
		zap.Int("daily_quota_remaining", remaining),
	)

	return c.JSON(response)
}

// GetInfo returns information about the faucet
func (h *Handler) GetInfo(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get faucet balances
	strkBalance, err := h.starknet.GetBalance(ctx, h.config.FaucetAddress, "STRK")
	if err != nil {
		h.logger.Error("Failed to get STRK balance", zap.Error(err))
		strkBalance = nil
	}

	ethBalance, err := h.starknet.GetBalance(ctx, h.config.FaucetAddress, "ETH")
	if err != nil {
		h.logger.Error("Failed to get ETH balance", zap.Error(err))
		ethBalance = nil
	}

	// Convert to readable format
	strkBalanceStr := "0"
	ethBalanceStr := "0"
	if strkBalance != nil {
		strkBalanceStr = fmt.Sprintf("%.2f", starknet.WeiToAmount(strkBalance))
	}
	if ethBalance != nil {
		ethBalanceStr = fmt.Sprintf("%.4f", starknet.WeiToAmount(ethBalance))
	}

	response := models.InfoResponse{
		Network: h.config.Network,
		Limits: models.LimitInfo{
			StrkPerRequest:     h.config.DripAmountSTRK,
			EthPerRequest:      h.config.DripAmountETH,
			DailyRequestsPerIP: h.config.MaxRequestsPerDayIP,
			TokenThrottleHours: 1, // 1 hour throttle per token
		},
		PoW: models.PoWInfo{
			Enabled:    true,
			Difficulty: h.config.PoWDifficulty,
		},
		FaucetBalance: models.BalanceInfo{
			STRK: strkBalanceStr,
			ETH:  ethBalanceStr,
		},
	}

	return c.JSON(response)
}

// handleBothTokensRequest handles requests for both STRK and ETH tokens
func (h *Handler) handleBothTokensRequest(c *fiber.Ctx, ctx context.Context, req models.FaucetRequest, ip string) error {
	// Process both STRK and ETH
	tokens := []string{"STRK", "ETH"}
	var transactions []models.TransactionInfo
	var failedToken string

	for _, token := range tokens {
		// Determine amount
		var amountStr string
		var amountFloat float64
		var maxHourly, maxDaily float64
		if token == "STRK" {
			amountStr = h.config.DripAmountSTRK
			amountFloat, _ = strconv.ParseFloat(amountStr, 64)
			maxHourly = h.config.MaxTokensPerHourSTRK
			maxDaily = h.config.MaxTokensPerDaySTRK
		} else {
			amountStr = h.config.DripAmountETH
			amountFloat, _ = strconv.ParseFloat(amountStr, 64)
			maxHourly = h.config.MaxTokensPerHourETH
			maxDaily = h.config.MaxTokensPerDayETH
		}

		// Check global distribution limits
		canDistribute, err := h.redis.TrackGlobalDistribution(ctx, token, amountFloat, maxHourly, maxDaily)
		if err != nil {
			h.logger.Error("Failed to check global distribution limits", zap.Error(err), zap.String("token", token))
			failedToken = token
			break
		}
		if !canDistribute {
			h.logger.Warn("Global distribution limit reached", zap.String("token", token), zap.String("ip", ip))
			failedToken = token
			break
		}

		// Check minimum balance protection
		currentBalance, err := h.starknet.GetBalance(ctx, h.config.FaucetAddress, token)
		if err != nil {
			h.logger.Error("Failed to check faucet balance", zap.Error(err), zap.String("token", token))
			failedToken = token
			break
		}

		amountWei := starknet.AmountToWei(amountFloat)
		minBalancePct := float64(h.config.MinBalanceProtectPct) / 100.0
		currentBalanceFloat := starknet.WeiToAmount(currentBalance)
		minBalanceRequired := currentBalanceFloat * minBalancePct
		balanceAfterTransfer := currentBalanceFloat - amountFloat

		if balanceAfterTransfer < minBalanceRequired {
			h.logger.Warn("Balance protection triggered", zap.String("token", token), zap.Float64("current_balance", currentBalanceFloat))
			failedToken = token
			break
		}

		// Transfer tokens
		h.logger.Info("Transferring tokens", zap.String("recipient", req.Address), zap.String("token", token), zap.String("amount", amountStr))

		txHash, err := h.starknet.TransferTokens(ctx, req.Address, token, amountWei)
		if err != nil {
			h.logger.Error("Failed to transfer tokens", zap.Error(err), zap.String("token", token))
			failedToken = token
			break
		}

		// Add to transactions list
		transactions = append(transactions, models.TransactionInfo{
			Token:       token,
			Amount:      amountStr,
			TxHash:      txHash,
			ExplorerURL: h.config.GetExplorerURL(txHash),
		})

		h.logger.Info("Tokens sent successfully", zap.String("tx_hash", txHash), zap.String("token", token))
	}

	// If any token failed and we have partial success, still return success with what worked
	if len(transactions) > 0 {
		// Increment IP daily counter by 2 (BOTH = 1 STRK + 1 ETH)
		if err := h.redis.IncrementIPDailyLimit(ctx, ip, 2); err != nil {
			h.logger.Error("Failed to increment IP daily limit", zap.Error(err))
		}

		// Set hourly throttle for both tokens
		for _, tx := range transactions {
			if err := h.redis.SetTokenHourlyThrottle(ctx, ip, tx.Token); err != nil {
				h.logger.Error("Failed to set token throttle", zap.Error(err), zap.String("token", tx.Token))
			}
		}

		message := "Both tokens sent successfully"
		if failedToken != "" {
			message = fmt.Sprintf("Sent %d token(s) successfully, but %s failed", len(transactions), failedToken)
		}

		return c.JSON(models.FaucetResponse{
			Success:      true,
			Transactions: transactions,
			Message:      message,
		})
	}

	// If no transactions succeeded, return error
	return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
		Error: fmt.Sprintf("Failed to send %s tokens. Please try again later.", failedToken),
	})
}

// GetQuota returns the current rate limit quota for the requesting IP
func (h *Handler) GetQuota(c *fiber.Ctx) error {
	ctx := context.Background()
	ip := c.IP()

	// Get IP daily quota
	used, remaining, cooldownEnd, err := h.redis.GetIPDailyQuota(ctx, ip)
	if err != nil {
		h.logger.Error("Failed to get IP daily quota", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to get quota",
		})
	}

	// Check token throttles
	strkThrottled, strkNext, err := h.redis.CheckTokenHourlyThrottle(ctx, ip, "STRK")
	if err != nil {
		h.logger.Error("Failed to check STRK throttle", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to check throttle",
		})
	}

	ethThrottled, ethNext, err := h.redis.CheckTokenHourlyThrottle(ctx, ip, "ETH")
	if err != nil {
		h.logger.Error("Failed to check ETH throttle", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to check throttle",
		})
	}

	response := map[string]interface{}{
		"daily_limit": map[string]interface{}{
			"total":              h.config.MaxRequestsPerDayIP,
			"used":               used,
			"remaining":          remaining,
			"cooldown_end":       cooldownEnd,
			"in_cooldown":        cooldownEnd != nil,
		},
		"hourly_throttle": map[string]interface{}{
			"strk": map[string]interface{}{
				"available":        strkThrottled,
				"next_request_at":  strkNext,
			},
			"eth": map[string]interface{}{
				"available":       ethThrottled,
				"next_request_at": ethNext,
			},
		},
	}

	return c.JSON(response)
}

// Health returns the health status of the API
func (h *Handler) Health(c *fiber.Ctx) error {
	ctx := context.Background()

	// Check Redis
	if err := h.redis.Ping(ctx); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.ErrorResponse{
			Error: "Redis unavailable",
		})
	}

	return c.JSON(models.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Unix(),
	})
}
