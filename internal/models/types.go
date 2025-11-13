package models

import "time"

// ChallengeRequest represents a request for a PoW challenge
type ChallengeRequest struct{}

// ChallengeResponse represents the response containing a PoW challenge
type ChallengeResponse struct {
	ChallengeID string `json:"challenge_id"`
	Challenge   string `json:"challenge"`
	Difficulty  int    `json:"difficulty"`
}

// FaucetRequest represents a request for tokens from the faucet
type FaucetRequest struct {
	Address     string `json:"address" validate:"required"`
	Token       string `json:"token" validate:"required,oneof=ETH STRK"`
	ChallengeID string `json:"challenge_id" validate:"required"`
	Nonce       int64  `json:"nonce" validate:"required"`
}

// FaucetResponse represents the successful response from a faucet request
type FaucetResponse struct {
	Success     bool   `json:"success"`
	TxHash      string `json:"tx_hash"`
	Amount      string `json:"amount"`
	Token       string `json:"token"`
	ExplorerURL string `json:"explorer_url"`
	Message     string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error           string     `json:"error"`
	NextRequestTime *time.Time `json:"next_request_time,omitempty"`
	RemainingHours  *float64   `json:"remaining_hours,omitempty"`
}

// StatusResponse represents the status of an address
type StatusResponse struct {
	Address         string     `json:"address"`
	CanRequest      bool       `json:"can_request"`
	LastRequest     *time.Time `json:"last_request,omitempty"`
	NextRequestTime *time.Time `json:"next_request_time,omitempty"`
	RemainingHours  *float64   `json:"remaining_hours,omitempty"`
}

// InfoResponse represents information about the faucet
type InfoResponse struct {
	Network      string         `json:"network"`
	Limits       LimitInfo      `json:"limits"`
	PoW          PoWInfo        `json:"pow"`
	FaucetBalance BalanceInfo   `json:"faucet_balance"`
}

// LimitInfo contains information about faucet limits
type LimitInfo struct {
	StrkPerRequest string `json:"strk_per_request"`
	EthPerRequest  string `json:"eth_per_request"`
	CooldownHours  int    `json:"cooldown_hours"`
}

// PoWInfo contains information about PoW requirements
type PoWInfo struct {
	Enabled    bool `json:"enabled"`
	Difficulty int  `json:"difficulty"`
}

// BalanceInfo contains information about faucet balances
type BalanceInfo struct {
	STRK string `json:"strk"`
	ETH  string `json:"eth"`
}

// HealthResponse represents the health status of the API
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}
