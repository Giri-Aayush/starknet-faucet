package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	apiURL  string
	verbose bool
	jsonOut bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "starknet-faucet",
	Short: "Starknet Sepolia Testnet Faucet CLI",
	Long: `A CLI tool to request testnet tokens (ETH and STRK) for Starknet Sepolia.

Commands:
  request <ADDRESS> [flags]  Request testnet tokens
  quota                      Check YOUR remaining quota (how many requests left)
  limits                     Show detailed rate limit rules
  status <ADDRESS>           Check request status
  info                       View faucet information

Examples:
  starknet-faucet request 0xYOUR_ADDRESS              # Request STRK tokens
  starknet-faucet request 0xYOUR_ADDRESS --token ETH  # Request ETH tokens
  starknet-faucet request 0xYOUR_ADDRESS --both       # Request both STRK and ETH
  starknet-faucet quota                               # Check YOUR remaining quota
  starknet-faucet limits                              # View rate limit rules
  starknet-faucet status 0xYOUR_ADDRESS               # Check status

Rate Limits (per IP):
  Daily Limit:
    • 5 requests per day
    • Single token (STRK or ETH) = 1 request
    • Both tokens (--both) = 2 requests
    • After 5th request: 24-hour cooldown

  Hourly Throttle:
    • 1 STRK request per hour
    • 1 ETH request per hour
    • Independent for each token

  Run 'starknet-faucet limits' for detailed examples

Security:
  • Proof of Work challenge (prevents bot abuse)
  • CAPTCHA verification (human check)

Need help? Visit: https://github.com/Giri-Aayush/starknet-faucet`,
	Version: "1.0.13",
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "https://starknet-faucet-gnq5.onrender.com", "Faucet API URL")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output in JSON format")

	// Add subcommands
	rootCmd.AddCommand(requestCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(limitsCmd)
	rootCmd.AddCommand(quotaCmd)
}
