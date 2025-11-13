package pow

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Giri-Aayush/starknet-faucet/internal/models"
)

// Challenge represents a PoW challenge
type Challenge struct {
	ID         string
	Challenge  string
	Difficulty int
	CreatedAt  time.Time
}

// Generator handles PoW challenge generation and verification
type Generator struct {
	difficulty int
	ttl        time.Duration
}

// NewGenerator creates a new PoW generator
func NewGenerator(difficulty int, ttlSeconds int) *Generator {
	return &Generator{
		difficulty: difficulty,
		ttl:        time.Duration(ttlSeconds) * time.Second,
	}
}

// GenerateChallenge creates a new PoW challenge
func (g *Generator) GenerateChallenge() (*models.ChallengeResponse, *Challenge, error) {
	// Generate random challenge string
	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		return nil, nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Generate challenge ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, nil, fmt.Errorf("failed to generate challenge ID: %w", err)
	}

	challenge := &Challenge{
		ID:         hex.EncodeToString(idBytes),
		Challenge:  hex.EncodeToString(challengeBytes),
		Difficulty: g.difficulty,
		CreatedAt:  time.Now(),
	}

	response := &models.ChallengeResponse{
		ChallengeID: challenge.ID,
		Challenge:   challenge.Challenge,
		Difficulty:  challenge.Difficulty,
	}

	return response, challenge, nil
}

// VerifyPoW verifies a PoW solution
func (g *Generator) VerifyPoW(challenge string, nonce int64, difficulty int) bool {
	// Ensure difficulty matches
	if difficulty != g.difficulty {
		return false
	}

	// Compute hash
	data := fmt.Sprintf("%s%d", challenge, nonce)
	hash := sha256.Sum256([]byte(data))
	hashHex := hex.EncodeToString(hash[:])

	// Check leading zeros
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hashHex, prefix)
}

// IsExpired checks if a challenge has expired
func (g *Generator) IsExpired(createdAt time.Time) bool {
	return time.Since(createdAt) > g.ttl
}

// SolveChallenge solves a PoW challenge (used by CLI)
func SolveChallenge(challenge string, difficulty int, progressCallback func(int64)) (int64, error) {
	prefix := strings.Repeat("0", difficulty)

	var nonce int64
	for {
		data := fmt.Sprintf("%s%d", challenge, nonce)
		hash := sha256.Sum256([]byte(data))
		hashHex := hex.EncodeToString(hash[:])

		if strings.HasPrefix(hashHex, prefix) {
			return nonce, nil
		}

		nonce++

		// Call progress callback every 10000 iterations
		if progressCallback != nil && nonce%10000 == 0 {
			progressCallback(nonce)
		}

		// Safety check - shouldn't happen with difficulty 4
		if nonce > 100000000 {
			return 0, fmt.Errorf("failed to solve challenge after %d attempts", nonce)
		}
	}
}

// EstimateSolveTime estimates how long it will take to solve a challenge
func EstimateSolveTime(difficulty int) time.Duration {
	// Rough estimate: 16^difficulty attempts on average
	// Assuming ~1M hashes per second on average CPU
	attempts := 1
	for i := 0; i < difficulty; i++ {
		attempts *= 16
	}

	// Assume 500k hashes per second (conservative)
	hashesPerSecond := 500000
	seconds := attempts / hashesPerSecond

	// Add 20% buffer
	seconds = int(float64(seconds) * 1.2)

	return time.Duration(seconds) * time.Second
}
