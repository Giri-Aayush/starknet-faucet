package commands

import (
	"encoding/json"
	"fmt"

	"github.com/Giri-Aayush/starknet-faucet/pkg/cli"
	"github.com/Giri-Aayush/starknet-faucet/pkg/cli/ui"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get faucet information",
	Long: `Get information about the faucet including limits, balances, and configuration.

Example:
  starknet-faucet info`,
	RunE: runInfo,
}

func runInfo(cmd *cobra.Command, args []string) error {
	// Create API client
	client := cli.NewAPIClient(apiURL)

	// Get info
	resp, err := client.GetInfo()
	if err != nil {
		return fmt.Errorf("failed to get info: %w", err)
	}

	// Print response
	if jsonOut {
		jsonBytes, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		ui.PrintBanner()
		ui.PrintInfoResponse(resp)
	}

	return nil
}
