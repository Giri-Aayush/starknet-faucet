package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStarknetAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{
			name:    "valid address - full length",
			address: "0x0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795f",
			wantErr: false,
		},
		{
			name:    "valid address - short",
			address: "0x123",
			wantErr: false,
		},
		{
			name:    "valid address - medium",
			address: "0x0742d469482a89e7dbbf139e872d4eeb",
			wantErr: false,
		},
		{
			name:    "empty address",
			address: "",
			wantErr: true,
		},
		{
			name:    "missing 0x prefix",
			address: "0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795f",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			address: "0x0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795g",
			wantErr: true,
		},
		{
			name:    "too long",
			address: "0x0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795f123",
			wantErr: true,
		},
		{
			name:    "just 0x",
			address: "0x",
			wantErr: true,
		},
		{
			name:    "uppercase hex",
			address: "0x0742D469482A89E7DBBF139E872D4EEB0F78DE5CC9962DE6EAEF71EF90E8795F",
			wantErr: false,
		},
		{
			name:    "mixed case hex",
			address: "0x0742d469482A89E7dbbf139e872D4eeb0f78de5cc9962de6eaef71ef90e8795F",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStarknetAddress(tt.address)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNormalizeStarknetAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "already full length",
			address:  "0x0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795f",
			expected: "0x0742d469482a89e7dbbf139e872d4eeb0f78de5cc9962de6eaef71ef90e8795f",
		},
		{
			name:     "short address",
			address:  "0x123",
			expected: "0x0000000000000000000000000000000000000000000000000000000000000123",
		},
		{
			name:     "medium address",
			address:  "0x0742d469482a89e7",
			expected: "0x00000000000000000000000000000000000000000000000000000742d469482a89e7",
		},
		{
			name:     "single digit",
			address:  "0x1",
			expected: "0x0000000000000000000000000000000000000000000000000000000000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeStarknetAddress(tt.address)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, result, 66) // 0x + 64 hex chars
		})
	}
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "ETH uppercase",
			token:   "ETH",
			wantErr: false,
		},
		{
			name:    "STRK uppercase",
			token:   "STRK",
			wantErr: false,
		},
		{
			name:    "eth lowercase",
			token:   "eth",
			wantErr: false,
		},
		{
			name:    "strk lowercase",
			token:   "strk",
			wantErr: false,
		},
		{
			name:    "mixed case",
			token:   "Eth",
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "BTC",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "random string",
			token:   "USDC",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
