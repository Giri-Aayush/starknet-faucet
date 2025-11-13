package pow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	difficulty := 4
	ttl := 300

	gen := NewGenerator(difficulty, ttl)

	assert.NotNil(t, gen)
	assert.Equal(t, difficulty, gen.difficulty)
	assert.Equal(t, time.Duration(ttl)*time.Second, gen.ttl)
}

func TestGenerateChallenge(t *testing.T) {
	gen := NewGenerator(4, 300)

	resp, challenge, err := gen.GenerateChallenge()

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, challenge)

	// Check response
	assert.NotEmpty(t, resp.ChallengeID)
	assert.NotEmpty(t, resp.Challenge)
	assert.Equal(t, 4, resp.Difficulty)

	// Check challenge
	assert.NotEmpty(t, challenge.ID)
	assert.NotEmpty(t, challenge.Challenge)
	assert.Equal(t, 4, challenge.Difficulty)
	assert.False(t, challenge.CreatedAt.IsZero())

	// Challenge ID should be 32 hex characters
	assert.Len(t, challenge.ID, 32)

	// Challenge should be 64 hex characters
	assert.Len(t, challenge.Challenge, 64)
}

func TestVerifyPoW(t *testing.T) {
	gen := NewGenerator(2, 300) // Use difficulty 2 for faster tests

	tests := []struct {
		name       string
		challenge  string
		nonce      int64
		difficulty int
		want       bool
	}{
		{
			name:       "valid solution",
			challenge:  "test123",
			nonce:      findValidNonce("test123", 2),
			difficulty: 2,
			want:       true,
		},
		{
			name:       "invalid solution",
			challenge:  "test123",
			nonce:      0,
			difficulty: 2,
			want:       false,
		},
		{
			name:       "wrong difficulty",
			challenge:  "test123",
			nonce:      findValidNonce("test123", 2),
			difficulty: 3,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.VerifyPoW(tt.challenge, tt.nonce, tt.difficulty)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsExpired(t *testing.T) {
	gen := NewGenerator(4, 1) // 1 second TTL

	tests := []struct {
		name      string
		createdAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			createdAt: time.Now(),
			want:      false,
		},
		{
			name:      "expired",
			createdAt: time.Now().Add(-2 * time.Second),
			want:      true,
		},
		{
			name:      "just expired",
			createdAt: time.Now().Add(-1 * time.Second),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.IsExpired(tt.createdAt)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSolveChallenge(t *testing.T) {
	challenge := "test123"
	difficulty := 2 // Use low difficulty for tests

	nonce, err := SolveChallenge(challenge, difficulty, nil)

	require.NoError(t, err)
	assert.Greater(t, nonce, int64(0))

	// Verify the solution
	gen := NewGenerator(difficulty, 300)
	assert.True(t, gen.VerifyPoW(challenge, nonce, difficulty))
}

func TestSolveChallengeWithCallback(t *testing.T) {
	challenge := "test456"
	difficulty := 2

	callbackCalled := false
	callback := func(n int64) {
		callbackCalled = true
		assert.Greater(t, n, int64(0))
	}

	nonce, err := SolveChallenge(challenge, difficulty, callback)

	require.NoError(t, err)
	assert.Greater(t, nonce, int64(0))
	assert.True(t, callbackCalled, "Callback should have been called")
}

func TestEstimateSolveTime(t *testing.T) {
	tests := []struct {
		name       string
		difficulty int
	}{
		{"difficulty 1", 1},
		{"difficulty 2", 2},
		{"difficulty 3", 3},
		{"difficulty 4", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimate := EstimateSolveTime(tt.difficulty)
			assert.Greater(t, estimate, time.Duration(0))
		})
	}
}

// Helper function to find a valid nonce for testing
func findValidNonce(challenge string, difficulty int) int64 {
	nonce, _ := SolveChallenge(challenge, difficulty, nil)
	return nonce
}

// Benchmark tests
func BenchmarkGenerateChallenge(b *testing.B) {
	gen := NewGenerator(4, 300)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = gen.GenerateChallenge()
	}
}

func BenchmarkVerifyPoW(b *testing.B) {
	gen := NewGenerator(2, 300)
	challenge := "test123"
	nonce := findValidNonce(challenge, 2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.VerifyPoW(challenge, nonce, 2)
	}
}

func BenchmarkSolveChallenge(b *testing.B) {
	challenge := "benchmark"
	difficulty := 2

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SolveChallenge(challenge, difficulty, nil)
	}
}
