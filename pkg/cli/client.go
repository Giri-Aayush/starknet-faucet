package cli

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/Giri-Aayush/starknet-faucet/internal/models"
)

// APIClient handles communication with the faucet API
type APIClient struct {
	baseURL string
	client  *resty.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	client := resty.New()
	client.SetTimeout(5 * time.Minute) // Long timeout for transaction waiting
	client.SetHeader("Content-Type", "application/json")

	return &APIClient{
		baseURL: baseURL,
		client:  client,
	}
}

// GetChallenge fetches a new PoW challenge
func (c *APIClient) GetChallenge() (*models.ChallengeResponse, error) {
	var response models.ChallengeResponse
	var errResponse models.ErrorResponse

	resp, err := c.client.R().
		SetResult(&response).
		SetError(&errResponse).
		Post(fmt.Sprintf("%s/api/v1/challenge", c.baseURL))

	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}

	if resp.IsError() {
		if errResponse.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResponse.Error)
		}
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode())
	}

	return &response, nil
}

// RequestTokens requests tokens from the faucet
func (c *APIClient) RequestTokens(req models.FaucetRequest) (*models.FaucetResponse, error) {
	var response models.FaucetResponse
	var errResponse models.ErrorResponse

	resp, err := c.client.R().
		SetBody(req).
		SetResult(&response).
		SetError(&errResponse).
		Post(fmt.Sprintf("%s/api/v1/faucet", c.baseURL))

	if err != nil {
		return nil, fmt.Errorf("failed to request tokens: %w", err)
	}

	if resp.IsError() {
		if errResponse.Error != "" {
			msg := errResponse.Error
			if errResponse.RemainingHours != nil {
				msg = fmt.Sprintf("%s (%.1f hours remaining)", msg, *errResponse.RemainingHours)
			}
			return nil, fmt.Errorf("API error: %s", msg)
		}
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode())
	}

	return &response, nil
}

// GetStatus checks the status of an address
func (c *APIClient) GetStatus(address string) (*models.StatusResponse, error) {
	var response models.StatusResponse
	var errResponse models.ErrorResponse

	resp, err := c.client.R().
		SetResult(&response).
		SetError(&errResponse).
		Get(fmt.Sprintf("%s/api/v1/status/%s", c.baseURL, address))

	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	if resp.IsError() {
		if errResponse.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResponse.Error)
		}
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode())
	}

	return &response, nil
}

// GetInfo gets information about the faucet
func (c *APIClient) GetInfo() (*models.InfoResponse, error) {
	var response models.InfoResponse
	var errResponse models.ErrorResponse

	resp, err := c.client.R().
		SetResult(&response).
		SetError(&errResponse).
		Get(fmt.Sprintf("%s/api/v1/info", c.baseURL))

	if err != nil {
		return nil, fmt.Errorf("failed to get info: %w", err)
	}

	if resp.IsError() {
		if errResponse.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResponse.Error)
		}
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode())
	}

	return &response, nil
}
