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

The faucet uses Proof of Work to prevent abuse and has a 24-hour cooldown
period per address.`,
	Version: "1.0.0",
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
}
