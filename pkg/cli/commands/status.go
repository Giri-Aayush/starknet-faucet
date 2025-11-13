package commands

import (
	"encoding/json"
	"fmt"

	"github.com/Giri-Aayush/starknet-faucet/pkg/cli"
	"github.com/Giri-Aayush/starknet-faucet/pkg/cli/ui"
	"github.com/Giri-Aayush/starknet-faucet/pkg/utils"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <ADDRESS>",
	Short: "Check cooldown status of an address",
	Long: `Check if an address is in cooldown period and when it can request tokens again.

Example:
  starknet-faucet status 0x0742...8d9f`,
	Args: cobra.ExactArgs(1),
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	address := args[0]

	// Validate address
	if err := utils.ValidateStarknetAddress(address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	// Create API client
	client := cli.NewAPIClient(apiURL)

	// Get status
	resp, err := client.GetStatus(address)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Print response
	if jsonOut {
		jsonBytes, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		ui.PrintBanner()
		ui.PrintStatusResponse(resp, address)
	}

	return nil
}
