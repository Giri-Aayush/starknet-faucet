package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Starknet address regex: 0x followed by up to 64 hex characters
	starknetAddressRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{1,64}$`)
)

// ValidateStarknetAddress validates a Starknet address format
func ValidateStarknetAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	if !strings.HasPrefix(address, "0x") {
		return fmt.Errorf("address must start with 0x")
	}

	if !starknetAddressRegex.MatchString(address) {
		return fmt.Errorf("invalid Starknet address format")
	}

	// Normalize to 66 characters (0x + 64 hex chars) by padding with zeros
	if len(address) < 66 {
		hexPart := address[2:]
		paddedHex := fmt.Sprintf("%064s", hexPart)
		paddedHex = strings.ReplaceAll(paddedHex, " ", "0")
		_ = "0x" + paddedHex
	}

	return nil
}

// NormalizeStarknetAddress normalizes a Starknet address to 66 characters
func NormalizeStarknetAddress(address string) string {
	if len(address) >= 66 {
		return address
	}

	hexPart := address[2:]
	paddedHex := fmt.Sprintf("%064s", hexPart)
	paddedHex = strings.ReplaceAll(paddedHex, " ", "0")
	return "0x" + paddedHex
}

// ValidateToken validates a token type
func ValidateToken(token string) error {
	token = strings.ToUpper(token)
	if token != "ETH" && token != "STRK" {
		return fmt.Errorf("invalid token: must be ETH or STRK")
	}
	return nil
}
