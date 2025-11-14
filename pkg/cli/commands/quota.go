package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Giri-Aayush/starknet-faucet/pkg/cli"
	"github.com/spf13/cobra"
)

var quotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Check your remaining rate limit quota",
	Long: `Display your current rate limit usage and remaining quota.

Shows:
  • Daily request quota (used/total)
  • STRK hourly throttle status
  • ETH hourly throttle status
  • Time until next available request

Example:
  starknet-faucet quota`,
	RunE: runQuota,
}

func runQuota(cmd *cobra.Command, args []string) error {
	// Create API client
	client := cli.NewAPIClient(apiURL)

	// Get quota
	resp, err := client.Get("/api/v1/quota")
	if err != nil {
		return fmt.Errorf("failed to get quota: %w", err)
	}

	// Parse response
	var quotaData map[string]interface{}
	if err := json.Unmarshal(resp, &quotaData); err != nil {
		return fmt.Errorf("failed to parse quota response: %w", err)
	}

	// Print response
	if jsonOut {
		jsonBytes, _ := json.MarshalIndent(quotaData, "", "  ")
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Pretty print quota
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              YOUR CURRENT RATE LIMIT QUOTA                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Daily limit
	dailyLimit := quotaData["daily_limit"].(map[string]interface{})
	total := int(dailyLimit["total"].(float64))
	used := int(dailyLimit["used"].(float64))
	remaining := int(dailyLimit["remaining"].(float64))

	fmt.Println("📊 DAILY QUOTA (Per IP)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Used:      %d/%d requests\n", used, total)
	fmt.Printf("  Remaining: %d requests\n", remaining)

	if remaining == 0 {
		fmt.Println("  ⚠️  Daily limit reached! Resets at midnight UTC")
	} else if remaining <= 2 {
		fmt.Printf("  ⚠️  Only %d request(s) left today\n", remaining)
	}
	fmt.Println()

	// Hourly throttles
	throttle := quotaData["hourly_throttle"].(map[string]interface{})
	strkData := throttle["strk"].(map[string]interface{})
	ethData := throttle["eth"].(map[string]interface{})

	strkAvailable := strkData["available"].(bool)
	ethAvailable := ethData["available"].(bool)

	fmt.Println("⏱  HOURLY THROTTLE STATUS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// STRK status
	if strkAvailable {
		fmt.Println("  STRK: ✅ Available now")
	} else {
		if nextTime := strkData["next_request_at"]; nextTime != nil {
			nextTimeStr := nextTime.(string)
			if nextT, err := time.Parse(time.RFC3339, nextTimeStr); err == nil {
				minutesLeft := int(time.Until(nextT).Minutes())
				if minutesLeft < 0 {
					minutesLeft = 0
				}
				fmt.Printf("  STRK: ⏳ Throttled (available in %d min)\n", minutesLeft)
			} else {
				fmt.Println("  STRK: ⏳ Throttled")
			}
		} else {
			fmt.Println("  STRK: ⏳ Throttled")
		}
	}

	// ETH status
	if ethAvailable {
		fmt.Println("  ETH:  ✅ Available now")
	} else {
		if nextTime := ethData["next_request_at"]; nextTime != nil {
			nextTimeStr := nextTime.(string)
			if nextT, err := time.Parse(time.RFC3339, nextTimeStr); err == nil {
				minutesLeft := int(time.Until(nextT).Minutes())
				if minutesLeft < 0 {
					minutesLeft = 0
				}
				fmt.Printf("  ETH:  ⏳ Throttled (available in %d min)\n", minutesLeft)
			} else {
				fmt.Println("  ETH:  ⏳ Throttled")
			}
		} else {
			fmt.Println("  ETH:  ⏳ Throttled")
		}
	}

	fmt.Println()

	// Recommendations
	if remaining > 0 {
		if strkAvailable && ethAvailable {
			fmt.Println("💡 You can request STRK or ETH tokens now")
			if remaining >= 2 {
				fmt.Println("   Or use --both to get both tokens (costs 2 requests)")
			}
		} else if strkAvailable {
			fmt.Println("💡 You can request STRK tokens now")
		} else if ethAvailable {
			fmt.Println("💡 You can request ETH tokens now")
		} else {
			fmt.Println("💡 Both tokens throttled. Please wait before requesting")
		}
	} else {
		fmt.Println("💡 Daily limit reached. Quota resets at midnight UTC")
	}

	fmt.Println()
	fmt.Println("Run 'starknet-faucet limits' to see detailed rate limit rules")
	fmt.Println()

	return nil
}
