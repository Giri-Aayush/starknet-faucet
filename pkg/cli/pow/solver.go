package pow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// SolveResult contains the result of solving a PoW challenge
type SolveResult struct {
	Nonce    int64
	Duration time.Duration
}

// Solver handles PoW challenge solving
type Solver struct{}

// NewSolver creates a new PoW solver
func NewSolver() *Solver {
	return &Solver{}
}

// Solve solves a PoW challenge with progress updates
func (s *Solver) Solve(challenge string, difficulty int, progressCallback func(int64, time.Duration)) (*SolveResult, error) {
	prefix := strings.Repeat("0", difficulty)
	startTime := time.Now()

	var nonce int64
	var lastUpdate time.Time

	for {
		data := fmt.Sprintf("%s%d", challenge, nonce)
		hash := sha256.Sum256([]byte(data))
		hashHex := hex.EncodeToString(hash[:])

		if strings.HasPrefix(hashHex, prefix) {
			// Found solution!
			duration := time.Since(startTime)
			return &SolveResult{
				Nonce:    nonce,
				Duration: duration,
			}, nil
		}

		nonce++

		// Call progress callback every 0.5 seconds
		if progressCallback != nil && time.Since(lastUpdate) >= 500*time.Millisecond {
			progressCallback(nonce, time.Since(startTime))
			lastUpdate = time.Now()
		}

		// Safety check - shouldn't happen with difficulty 4
		if nonce > 100000000 {
			return nil, fmt.Errorf("failed to solve challenge after %d attempts", nonce)
		}
	}
}

// EstimateSolveTime estimates how long it will take to solve a challenge
func EstimateSolveTime(difficulty int) time.Duration {
	// Rough estimate: 16^difficulty attempts on average
	attempts := 1
	for i := 0; i < difficulty; i++ {
		attempts *= 16
	}

	// Assume 500k hashes per second (conservative)
	hashesPerSecond := 500000
	seconds := attempts / hashesPerSecond

	// Add 20% buffer
	seconds = int(float64(seconds) * 1.2)

	if seconds < 1 {
		seconds = 1
	}

	return time.Duration(seconds) * time.Second
}
