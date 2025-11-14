package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	Port     string
	LogLevel string
	Network  string

	// Starknet
	FaucetPrivateKey string
	FaucetAddress    string
	StarknetRPCURL   string
	ETHTokenAddress  string
	STRKTokenAddress string

	// Redis
	RedisURL string

	// Faucet Settings
	PoWDifficulty   int
	DripAmountSTRK  string
	DripAmountETH   string
	ChallengeTTL    int // in seconds

	// Rate Limiting (Simplified)
	MaxRequestsPerDayIP  int // Max requests per IP per day (5) - single token=1, BOTH=2
	MaxChallengesPerHour int // Max PoW challenges per IP per hour (8)

	// Global Distribution Limits (prevents drain attacks)
	MaxTokensPerHourSTRK  float64 // Max STRK distributed per hour globally
	MaxTokensPerDaySTRK   float64 // Max STRK per day globally
	MaxTokensPerHourETH   float64 // Max ETH distributed per hour globally
	MaxTokensPerDayETH    float64 // Max ETH per day globally
	MinBalanceProtectPct  int     // Stop distributing when balance drops to this % (e.g., 20 = stop at 20%)
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (optional)
	_ = godotenv.Load()

	config := &Config{
		// Server defaults
		Port:     getEnv("PORT", "3000"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Network:  getEnv("NETWORK", "sepolia"),

		// Starknet (required)
		FaucetPrivateKey: getEnv("FAUCET_PRIVATE_KEY", ""),
		FaucetAddress:    getEnv("FAUCET_ADDRESS", ""),
		StarknetRPCURL:   getEnv("STARKNET_RPC_URL", ""),

		// Token addresses - Sepolia defaults
		ETHTokenAddress:  getEnv("ETH_TOKEN_ADDRESS", "0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7"),
		STRKTokenAddress: getEnv("STRK_TOKEN_ADDRESS", "0x04718f5a0fc34cc1af16a1cdee98ffb20c31f5cd61d6ab07201858f4287c938d"),

		// Redis (required)
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		// Faucet settings
		PoWDifficulty:  getEnvAsInt("POW_DIFFICULTY", 4),
		DripAmountSTRK: getEnv("DRIP_AMOUNT_STRK", "10"),
		DripAmountETH:  getEnv("DRIP_AMOUNT_ETH", "0.01"),
		ChallengeTTL:   getEnvAsInt("CHALLENGE_TTL", 300), // 5 minutes

		// Rate limiting (simplified)
		MaxRequestsPerDayIP:  getEnvAsInt("MAX_REQUESTS_PER_DAY_IP", 5), // 5 requests/day per IP
		MaxChallengesPerHour: getEnvAsInt("MAX_CHALLENGES_PER_HOUR", 8), // 8 challenges/hour per IP

		// Global distribution limits (anti-drain protection) - set to 0 to disable
		MaxTokensPerHourSTRK: getEnvAsFloat("MAX_TOKENS_PER_HOUR_STRK", 0), // 0 = disabled
		MaxTokensPerDaySTRK:  getEnvAsFloat("MAX_TOKENS_PER_DAY_STRK", 0),  // 0 = disabled
		MaxTokensPerHourETH:  getEnvAsFloat("MAX_TOKENS_PER_HOUR_ETH", 0),  // 0 = disabled
		MaxTokensPerDayETH:   getEnvAsFloat("MAX_TOKENS_PER_DAY_ETH", 0),   // 0 = disabled
		MinBalanceProtectPct: getEnvAsInt("MIN_BALANCE_PROTECT_PCT", 5),    // Stop at 5% remaining
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks if all required configuration is present
func (c *Config) Validate() error {
	if c.FaucetPrivateKey == "" {
		return fmt.Errorf("FAUCET_PRIVATE_KEY is required")
	}
	if c.FaucetAddress == "" {
		return fmt.Errorf("FAUCET_ADDRESS is required")
	}
	if c.StarknetRPCURL == "" {
		return fmt.Errorf("STARKNET_RPC_URL is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	return nil
}

// GetExplorerURL returns the block explorer URL for the configured network
func (c *Config) GetExplorerURL(txHash string) string {
	if c.Network == "mainnet" {
		return fmt.Sprintf("https://voyager.online/tx/%s", txHash)
	}
	return fmt.Sprintf("https://sepolia.voyager.online/tx/%s", txHash)
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}
