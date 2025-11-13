package starknet

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/NethermindEth/juno/core/felt"
	"github.com/NethermindEth/starknet.go/account"
	"github.com/NethermindEth/starknet.go/rpc"
	"github.com/NethermindEth/starknet.go/utils"
)

// FaucetClient handles Starknet blockchain interactions
type FaucetClient struct {
	account     *account.Account
	provider    *rpc.Provider
	ethAddress  *felt.Felt
	strkAddress *felt.Felt
}

// NewFaucetClient creates a new Starknet faucet client
func NewFaucetClient(rpcURL, privateKey, accountAddress, ethTokenAddr, strkTokenAddr string) (*FaucetClient, error) {
	ctx := context.Background()

	// Initialize RPC provider
	provider, err := rpc.NewProvider(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Parse private key
	privKeyBI, ok := new(big.Int).SetString(privateKey, 0)
	if !ok {
		return nil, fmt.Errorf("invalid private key format")
	}

	// Setup keystore
	ks := account.NewMemKeystore()
	ks.Put(accountAddress, privKeyBI)

	// Parse account address
	accAddress, err := utils.HexToFelt(accountAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid account address: %w", err)
	}

	// Create account (Cairo 2 - latest version)
	accnt, err := account.NewAccount(provider, accAddress, accountAddress, ks, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Parse token addresses
	ethAddr, err := utils.HexToFelt(ethTokenAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid ETH token address: %w", err)
	}

	strkAddr, err := utils.HexToFelt(strkTokenAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid STRK token address: %w", err)
	}

	return &FaucetClient{
		account:     accnt,
		provider:    provider,
		ethAddress:  ethAddr,
		strkAddress: strkAddr,
	}, nil
}

// TransferTokens transfers tokens to a recipient
func (fc *FaucetClient) TransferTokens(
	ctx context.Context,
	recipient string,
	token string,
	amount *big.Int,
) (string, error) {
	// Parse recipient address
	recipientFelt, err := utils.HexToFelt(recipient)
	if err != nil {
		return "", fmt.Errorf("invalid recipient address: %w", err)
	}

	// Determine token address
	var tokenAddress *felt.Felt
	switch token {
	case "ETH":
		tokenAddress = fc.ethAddress
	case "STRK":
		tokenAddress = fc.strkAddress
	default:
		return "", fmt.Errorf("invalid token: %s", token)
	}

	// Convert amount to Cairo uint256 format (low, high)
	low := new(big.Int).And(amount, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1)))
	high := new(big.Int).Rsh(amount, 128)

	lowFelt := new(felt.Felt).SetBigInt(low)
	highFelt := new(felt.Felt).SetBigInt(high)

	// Build transfer call
	call := rpc.InvokeFunctionCall{
		ContractAddress: tokenAddress,
		FunctionName:    "transfer",
		CallData: []*felt.Felt{
			recipientFelt,
			lowFelt,
			highFelt,
		},
	}

	// Build and send invoke transaction
	tx, err := fc.account.BuildAndSendInvokeTxn(ctx, []rpc.InvokeFunctionCall{call}, nil)
	if err != nil {
		return "", fmt.Errorf("transaction failed: %w", err)
	}

	// Return transaction hash
	return tx.Hash.String(), nil
}

// GetBalance gets the token balance of an address
func (fc *FaucetClient) GetBalance(ctx context.Context, address string, token string) (*big.Int, error) {
	// Parse address
	addrFelt, err := utils.HexToFelt(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	// Determine token address
	var tokenAddress *felt.Felt
	switch token {
	case "ETH":
		tokenAddress = fc.ethAddress
	case "STRK":
		tokenAddress = fc.strkAddress
	default:
		return nil, fmt.Errorf("invalid token: %s", token)
	}

	// Call balanceOf
	balanceSelector := utils.GetSelectorFromNameFelt("balanceOf")

	result, err := fc.provider.Call(ctx, rpc.FunctionCall{
		ContractAddress:    tokenAddress,
		EntryPointSelector: balanceSelector,
		Calldata:           []*felt.Felt{addrFelt},
	}, rpc.BlockID{Tag: "latest"})

	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("unexpected balance result length")
	}

	// Convert from uint256 (low, high) to big.Int
	low := result[0].BigInt(big.NewInt(0))
	high := result[1].BigInt(big.NewInt(0))

	balance := new(big.Int).Add(
		low,
		new(big.Int).Lsh(high, 128),
	)

	return balance, nil
}

// WaitForTransaction waits for a transaction to be accepted
func (fc *FaucetClient) WaitForTransaction(ctx context.Context, txHash string) error {
	txHashFelt, err := utils.HexToFelt(txHash)
	if err != nil {
		return fmt.Errorf("invalid tx hash: %w", err)
	}

	// Poll for transaction receipt
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check transaction receipt
			receipt, err := fc.provider.TransactionReceipt(ctx, txHashFelt)
			if err != nil {
				continue
			}

			// Check if transaction is accepted
			if receipt != nil {
				return nil
			}
		}
	}
}

// AmountToWei converts a float amount to wei (10^18)
func AmountToWei(amount float64) *big.Int {
	// 1 token = 10^18 wei
	weiPerToken := new(big.Float).SetInt(new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(18),
		nil,
	))

	amountFloat := new(big.Float).Mul(
		big.NewFloat(amount),
		weiPerToken,
	)

	amountInt, _ := amountFloat.Int(nil)
	return amountInt
}

// WeiToAmount converts wei to a float amount
func WeiToAmount(wei *big.Int) float64 {
	weiPerToken := new(big.Float).SetInt(new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(18),
		nil,
	))

	weiFloat := new(big.Float).SetInt(wei)
	amount := new(big.Float).Quo(weiFloat, weiPerToken)

	result, _ := amount.Float64()
	return result
}
