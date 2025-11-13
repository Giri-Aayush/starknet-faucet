package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/Giri-Aayush/starknet-faucet/internal/models"
)

var (
	// Colors
	cyan    = color.New(color.FgCyan).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	bold    = color.New(color.Bold).SprintFunc()

	// Symbols
	checkMark = green("✓")
	xMark     = red("✗")
	arrow     = cyan("→")
)

// PrintBanner prints the faucet banner
func PrintBanner() {
	banner := `
   ███████╗████████╗ █████╗ ██████╗ ██╗  ██╗███╗   ██╗███████╗████████╗    ███████╗ █████╗ ██╗   ██╗ ██████╗███████╗████████╗
   ██╔════╝╚══██╔══╝██╔══██╗██╔══██╗██║ ██╔╝████╗  ██║██╔════╝╚══██╔══╝    ██╔════╝██╔══██╗██║   ██║██╔════╝██╔════╝╚══██╔══╝
   ███████╗   ██║   ███████║██████╔╝█████╔╝ ██╔██╗ ██║█████╗     ██║       █████╗  ███████║██║   ██║██║     █████╗     ██║
   ╚════██║   ██║   ██╔══██║██╔══██╗██╔═██╗ ██║╚██╗██║██╔══╝     ██║       ██╔══╝  ██╔══██║██║   ██║██║     ██╔══╝     ██║
   ███████║   ██║   ██║  ██║██║  ██║██║  ██╗██║ ╚████║███████╗   ██║       ██║     ██║  ██║╚██████╔╝╚██████╗███████╗   ██║
   ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝       ╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚═════╝╚══════╝   ╚═╝

                                         Made with love • Secured by PoW
`
	fmt.Println(cyan(banner))
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Printf("%s %s\n", checkMark, message)
}

// PrintError prints an error message
func PrintError(message string) {
	fmt.Printf("%s %s\n", xMark, red(message))
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	fmt.Printf("%s %s\n", arrow, message)
}

// NewSpinner creates a new spinner with a message
func NewSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Color("cyan")
	return s
}

// PrintFaucetResponse prints a nicely formatted faucet response
func PrintFaucetResponse(resp *models.FaucetResponse) {
	fmt.Println()
	fmt.Println(strings.Repeat("━", 50))
	fmt.Printf("  %s  %s %s\n", bold("Amount:"), resp.Amount, resp.Token)
	fmt.Printf("  %s  %s\n", bold("TX Hash:"), shortenHash(resp.TxHash))
	fmt.Println()
	fmt.Printf("  🔗 %s\n", cyan(resp.ExplorerURL))
	fmt.Println(strings.Repeat("━", 50))
	fmt.Println()
	PrintSuccess("Tokens will arrive in ~30 seconds.")
	fmt.Println()
}

// PrintStatusResponse prints a status response
func PrintStatusResponse(resp *models.StatusResponse, address string) {
	fmt.Println()
	fmt.Printf("%s %s\n\n", bold("Address:"), shortenHash(address))

	if resp.CanRequest {
		PrintSuccess("This address can request tokens now!")
	} else {
		PrintError("Address is in cooldown period")
		fmt.Println()
		if resp.LastRequest != nil {
			fmt.Printf("  Last request:  %s\n", resp.LastRequest.Format("January 02, 2006 at 3:04 PM"))
		}
		if resp.NextRequestTime != nil {
			fmt.Printf("  Next request:  %s\n", resp.NextRequestTime.Format("January 02, 2006 at 3:04 PM"))
		}
		if resp.RemainingHours != nil {
			fmt.Printf("  Time remaining: %s\n", formatDuration(*resp.RemainingHours))
		}
	}
	fmt.Println()
}

// PrintInfoResponse prints an info response
func PrintInfoResponse(resp *models.InfoResponse) {
	fmt.Println()
	fmt.Println(bold("Faucet Information"))
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()

	fmt.Printf("%s %s\n", bold("Network:"), resp.Network)
	fmt.Println()

	fmt.Println(bold("Distribution Limits:"))
	fmt.Printf("  STRK per request:  %s STRK\n", resp.Limits.StrkPerRequest)
	fmt.Printf("  ETH per request:   %s ETH\n", resp.Limits.EthPerRequest)
	fmt.Printf("  Cooldown period:   %d hours\n", resp.Limits.CooldownHours)
	fmt.Println()

	fmt.Println(bold("Proof of Work:"))
	fmt.Printf("  Enabled:    %v\n", resp.PoW.Enabled)
	fmt.Printf("  Difficulty: %d\n", resp.PoW.Difficulty)
	fmt.Println()

	fmt.Println(bold("Faucet Balance:"))
	fmt.Printf("  STRK: %s\n", resp.FaucetBalance.STRK)
	fmt.Printf("  ETH:  %s\n", resp.FaucetBalance.ETH)
	fmt.Println()
}

// PrintCooldownError prints a cooldown error with details
func PrintCooldownError(nextRequestTime *time.Time, remainingHours *float64) {
	fmt.Println()
	PrintError("Address is in cooldown period")
	fmt.Println()
	if nextRequestTime != nil {
		fmt.Printf("  Next request:  %s\n", nextRequestTime.Format("January 02, 2006 at 3:04 PM"))
	}
	if remainingHours != nil {
		fmt.Printf("  Time remaining: %s\n", formatDuration(*remainingHours))
	}
	fmt.Println()
	fmt.Println("Try again later or use --help for more options.")
	fmt.Println()
}

// Helper functions

func shortenHash(hash string) string {
	if len(hash) <= 20 {
		return hash
	}
	return hash[:10] + "..." + hash[len(hash)-8:]
}

func formatDuration(hours float64) string {
	if hours >= 24 {
		days := int(hours / 24)
		remainingHours := int(hours) % 24
		if remainingHours == 0 {
			return fmt.Sprintf("%d day%s", days, pluralize(days))
		}
		return fmt.Sprintf("%d day%s %d hour%s", days, pluralize(days), remainingHours, pluralize(remainingHours))
	}

	if hours >= 1 {
		h := int(hours)
		minutes := int((hours - float64(h)) * 60)
		if minutes == 0 {
			return fmt.Sprintf("%d hour%s", h, pluralize(h))
		}
		return fmt.Sprintf("%d hour%s %d minute%s", h, pluralize(h), minutes, pluralize(minutes))
	}

	minutes := int(hours * 60)
	return fmt.Sprintf("%d minute%s", minutes, pluralize(minutes))
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
