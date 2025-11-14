package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Giri-Aayush/starknet-faucet/internal/models"
	"github.com/Giri-Aayush/starknet-faucet/pkg/cli"
	"github.com/Giri-Aayush/starknet-faucet/pkg/cli/captcha"
	clipow "github.com/Giri-Aayush/starknet-faucet/pkg/cli/pow"
	"github.com/Giri-Aayush/starknet-faucet/pkg/cli/ui"
	"github.com/Giri-Aayush/starknet-faucet/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	token string
	both  bool
)

var requestCmd = &cobra.Command{
	Use:   "request <ADDRESS>",
	Short: "Request testnet tokens",
	Long: `Request testnet tokens (ETH or STRK) for a Starknet address.

The faucet distributes:
  • 10 STRK per request
  • 0.01 ETH per request

Rate Limits:
  Individual tokens: 2/hour, 3/day per token
  Both tokens:       1/day (24-hour cooldown)

Examples:
  # Request STRK tokens (default)
  starknet-faucet request 0x0742...8d9f

  # Request ETH tokens
  starknet-faucet request 0x0742...8d9f --token ETH

  # Request both tokens (10 STRK + 0.01 ETH in single transaction)
  starknet-faucet request 0x0742...8d9f --both
  starknet-faucet request 0x0742...8d9f --token both

Security:
  Each request requires:
  • Proof of Work challenge (computational work)
  • CAPTCHA verification (human check)

Note: Using --both counts toward your individual token limits
      AND sets a 24-hour cooldown for --both requests.`,
	Args: cobra.ExactArgs(1),
	RunE: runRequest,
}

func init() {
	requestCmd.Flags().StringVar(&token, "token", "STRK", "Token to request (ETH or STRK)")
	requestCmd.Flags().BoolVar(&both, "both", false, "Request both ETH and STRK")
}

func runRequest(cmd *cobra.Command, args []string) error {
	address := args[0]

	// Validate address
	if err := utils.ValidateStarknetAddress(address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	// Normalize token
	token = strings.ToUpper(token)

	// Handle "both" as token value
	if token == "BOTH" {
		both = true
	}

	// Validate token (if not requesting both)
	if !both {
		if err := utils.ValidateToken(token); err != nil {
			return err
		}
	}

	// Create API client
	client := cli.NewAPIClient(apiURL)

	// Print banner (unless JSON output)
	if !jsonOut {
		ui.PrintBanner()

		// Ask verification question (3 attempts)
		correct, err := captcha.AskQuestionWithRetries(3)
		if err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}
		if !correct {
			return fmt.Errorf("verification failed - please try again later")
		}
	}

	// Request tokens
	if both {
		// Request STRK first, then ETH
		if err := requestSingleToken(client, address, "STRK"); err != nil {
			return err
		}
		fmt.Println() // Add spacing
		if err := requestSingleToken(client, address, "ETH"); err != nil {
			return err
		}
	} else {
		if err := requestSingleToken(client, address, token); err != nil {
			return err
		}
	}

	return nil
}

func requestSingleToken(client *cli.APIClient, address, token string) error {
	if !jsonOut {
		ui.PrintInfo(fmt.Sprintf("Requesting %s for %s", token, address))
		fmt.Println()
	}

	// Step 1: Get challenge
	var challengeResp *models.ChallengeResponse
	if !jsonOut {
		s := ui.NewSpinner("Fetching challenge...")
		s.Start()
		var err error
		challengeResp, err = client.GetChallenge()
		s.Stop()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get challenge: %v", err))
			return err
		}
		ui.PrintSuccess("Challenge received")
		fmt.Println()
	} else {
		var err error
		challengeResp, err = client.GetChallenge()
		if err != nil {
			return err
		}
	}

	// Step 2: Solve PoW
	var nonce int64
	var solveDuration time.Duration
	if !jsonOut {
		s := ui.NewSpinner(fmt.Sprintf("Solving proof of work (difficulty: %d)...", challengeResp.Difficulty))
		s.Start()

		solver := clipow.NewSolver()
		result, err := solver.Solve(challengeResp.Challenge, challengeResp.Difficulty, func(n int64, d time.Duration) {
			// Update spinner suffix with progress
			s.Suffix = fmt.Sprintf(" Solving proof of work (attempts: %d, time: %.1fs)...",
				n, d.Seconds())
		})

		s.Stop()

		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to solve challenge: %v", err))
			return err
		}

		nonce = result.Nonce
		solveDuration = result.Duration
		ui.PrintSuccess(fmt.Sprintf("Challenge solved in %.1fs (nonce: %d)", solveDuration.Seconds(), nonce))
		fmt.Println()
	} else {
		solver := clipow.NewSolver()
		result, err := solver.Solve(challengeResp.Challenge, challengeResp.Difficulty, nil)
		if err != nil {
			return err
		}
		nonce = result.Nonce
		solveDuration = result.Duration
	}

	// Step 3: Request tokens
	req := models.FaucetRequest{
		Address:     address,
		Token:       token,
		ChallengeID: challengeResp.ChallengeID,
		Nonce:       nonce,
	}

	var faucetResp *models.FaucetResponse
	if !jsonOut {
		s := ui.NewSpinner("Submitting request...")
		s.Start()
		var err error
		faucetResp, err = client.RequestTokens(req)
		s.Stop()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to request tokens: %v", err))
			return err
		}
		ui.PrintSuccess("Transaction submitted!")
	} else {
		var err error
		faucetResp, err = client.RequestTokens(req)
		if err != nil {
			return err
		}
	}

	// Print response
	if jsonOut {
		output := map[string]interface{}{
			"success":        faucetResp.Success,
			"tx_hash":        faucetResp.TxHash,
			"amount":         faucetResp.Amount,
			"token":          faucetResp.Token,
			"explorer_url":   faucetResp.ExplorerURL,
			"solve_duration": solveDuration.Seconds(),
		}
		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		ui.PrintFaucetResponse(faucetResp)
	}

	return nil
}
