package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var limitsCmd = &cobra.Command{
	Use:   "limits",
	Short: "Show rate limit information",
	Long: `Display detailed rate limiting rules for the faucet.

Learn about daily limits, hourly throttles, and request costs.

Example:
  starknet-faucet limits`,
	RunE: runLimits,
}

func runLimits(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║            STARKNET FAUCET RATE LIMITS                        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Println("📊 DAILY LIMIT (Per IP)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  • 5 requests per day")
	fmt.Println("  • Single token (STRK or ETH) = 1 request")
	fmt.Println("  • Both tokens (--both) = 2 requests (1 STRK + 1 ETH)")
	fmt.Println("  • After 5th request: 24-hour cooldown")
	fmt.Println("  • Cooldown starts from the time of 5th request")
	fmt.Println()

	fmt.Println("⏱  HOURLY THROTTLE (Per Token)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  • 1 STRK request per hour")
	fmt.Println("  • 1 ETH request per hour")
	fmt.Println("  • Independent for each token")
	fmt.Println()

	fmt.Println("💡 EXAMPLES")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("  Example 1: Requesting same token multiple times")
	fmt.Println("  ─────────────────────────────────────────────────")
	fmt.Println("  10:00 AM → Request STRK ✓ (1/5 daily)")
	fmt.Println("  10:30 AM → Request STRK ✗ (throttled - wait 30 min)")
	fmt.Println("  11:00 AM → Request STRK ✓ (2/5 daily)")
	fmt.Println()

	fmt.Println("  Example 2: Requesting different tokens")
	fmt.Println("  ───────────────────────────────────────")
	fmt.Println("  10:00 AM → Request STRK ✓ (1/5 daily)")
	fmt.Println("  10:01 AM → Request ETH  ✓ (2/5 daily, different token)")
	fmt.Println("  11:00 AM → Request STRK ✓ (3/5 daily)")
	fmt.Println("  11:01 AM → Request ETH  ✓ (4/5 daily)")
	fmt.Println()

	fmt.Println("  Example 3: Using --both flag")
	fmt.Println("  ────────────────────────────")
	fmt.Println("  10:00 AM → Request --both ✓ (2/5 daily, both throttled)")
	fmt.Println("  10:30 AM → Request STRK   ✗ (throttled - wait 30 min)")
	fmt.Println("  10:30 AM → Request ETH    ✗ (throttled - wait 30 min)")
	fmt.Println("  11:00 AM → Request STRK   ✓ (3/5 daily)")
	fmt.Println("  12:00 PM → Request ETH    ✓ (4/5 daily)")
	fmt.Println()

	fmt.Println("📋 ADDITIONAL LIMITS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  • PoW challenges: 8 per hour (allows 3 failures per token)")
	fmt.Println("  • CAPTCHA verification required for each request")
	fmt.Println()

	fmt.Println("💰 AMOUNTS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  • STRK: 10 STRK per request")
	fmt.Println("  • ETH:  0.01 ETH per request")
	fmt.Println()

	return nil
}
