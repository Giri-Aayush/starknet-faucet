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

	// Check IP rate limit
	ip := c.IP()
	canRequest, err := h.redis.CheckIPRateLimit(ctx, ip)
	if err != nil {
		h.logger.Error("Failed to check IP rate limit", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to check rate limit",
		})
	}
	if !canRequest {
		return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
			Error: "IP rate limit exceeded",
		})
	}

	// Check address cooldown
	inCooldown, nextRequestTime, err := h.redis.IsAddressInCooldown(ctx, req.Address)
	if err != nil {
		h.logger.Error("Failed to check address cooldown", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to check cooldown",
		})
	}
	if inCooldown {
		remainingHours := time.Until(*nextRequestTime).Hours()
		return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
			Error:           "Address in cooldown period",
			NextRequestTime: nextRequestTime,
			RemainingHours:  &remainingHours,
		})
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

	// Determine amount
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

	// Set cooldown and increment rate limit
	if err := h.redis.SetAddressCooldown(ctx, req.Address); err != nil {
		h.logger.Error("Failed to set cooldown", zap.Error(err))
	}
	if err := h.redis.IncrementIPRateLimit(ctx, ip); err != nil {
		h.logger.Error("Failed to increment rate limit", zap.Error(err))
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

	// Check cooldown
	inCooldown, nextRequestTime, err := h.redis.IsAddressInCooldown(ctx, address)
	if err != nil {
		h.logger.Error("Failed to check cooldown", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "Failed to check status",
		})
	}

	response := models.StatusResponse{
		Address:    address,
		CanRequest: !inCooldown,
	}

	if inCooldown {
		remainingHours := time.Until(*nextRequestTime).Hours()
		lastRequest := nextRequestTime.Add(-time.Duration(h.config.CooldownHours) * time.Hour)
		response.LastRequest = &lastRequest
		response.NextRequestTime = nextRequestTime
		response.RemainingHours = &remainingHours
	}

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
			StrkPerRequest: h.config.DripAmountSTRK,
			EthPerRequest:  h.config.DripAmountETH,
			CooldownHours:  h.config.CooldownHours,
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
