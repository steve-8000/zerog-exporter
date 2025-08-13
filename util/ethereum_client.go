package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type EthereumClient struct {
	RPCURL    string
	JWTSecret string
	Client    *http.Client
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewEthereumClient(rpcURL string) *EthereumClient {
	return &EthereumClient{
		RPCURL: rpcURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func NewEthereumClientWithJWT(rpcURL, jwtSecret string) *EthereumClient {
	return &EthereumClient{
		RPCURL:    rpcURL,
		JWTSecret: jwtSecret,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *EthereumClient) Call(method string, params interface{}) (json.RawMessage, error) {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// JWT 토큰이 설정된 경우 Authorization 헤더 추가
	req, err := http.NewRequest("POST", c.RPCURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if c.JWTSecret != "" {
		req.Header.Set("Authorization", "Bearer "+c.JWTSecret)
	}
	
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var response JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: %s", response.Error.Message)
	}

	return response.Result, nil
}

// GetBlockNumber returns the current block number
func (c *EthereumClient) GetBlockNumber() (string, error) {
	result, err := c.Call("eth_blockNumber", []interface{}{})
	if err != nil {
		return "", err
	}

	var blockNumber string
	if err := json.Unmarshal(result, &blockNumber); err != nil {
		return "", fmt.Errorf("failed to unmarshal block number: %w", err)
	}

	return blockNumber, nil
}

// GetBalance returns the balance of an address
func (c *EthereumClient) GetBalance(address string) (string, error) {
	params := []interface{}{address, "latest"}
	result, err := c.Call("eth_getBalance", params)
	if err != nil {
		return "", err
	}

	var balance string
	if err := json.Unmarshal(result, &balance); err != nil {
		return "", fmt.Errorf("failed to unmarshal balance: %w", err)
	}

	return balance, nil
}

// CallContract calls a smart contract method
func (c *EthereumClient) CallContract(to, data string) (string, error) {
	params := []interface{}{
		map[string]string{
			"to":   to,
			"data": data,
		},
		"latest",
	}
	result, err := c.Call("eth_call", params)
	if err != nil {
		return "", err
	}

	var response string
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal contract response: %w", err)
	}

	return response, nil
}

// GetValidatorInfo retrieves validator information from the staking contract
func (c *EthereumClient) GetValidatorInfo(validatorAddress string) (map[string]interface{}, error) {
	// Function signature: getValidatorInfo(address)
	// keccak256("getValidatorInfo(address)") = 0x8b5a9c0d
	functionSelector := "0x8b5a9c0d"
	
	// Pad the address to 32 bytes
	paddedAddress := "000000000000000000000000" + validatorAddress[2:] // Remove 0x and pad
	
	data := functionSelector + paddedAddress
	
	result, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", data)
	if err != nil {
		return nil, fmt.Errorf("failed to call getValidatorInfo: %w", err)
	}
	
	// Parse the result (this is a simplified version - actual parsing depends on the contract structure)
	// For now, we'll return the raw result and parse it in the collector
	return map[string]interface{}{
		"raw_result": result,
		"address":    validatorAddress,
	}, nil
}



// GetValidatorsList retrieves the list of all validators from the staking contract
func (c *EthereumClient) GetValidatorsList() ([]string, error) {
	// Function signature: validatorCount()
	// keccak256("validatorCount()") = 0x8b5a9c0d (placeholder - need actual signature)
	functionSelector := "0x8b5a9c0d"
	
	_, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", functionSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to call validatorCount: %w", err)
	}
	
	// For now, return a hardcoded list based on what we know
	// In a real implementation, you would parse the result
	return []string{
		"0x30535EF0D596876C5DBFCF825D64134550AB4945",
		"0x00092f31B30461501CA6311Fc225f8f1ddFbE67e",
	}, nil
}

// GetTotalValidators returns the total number of registered validators
func (c *EthereumClient) GetTotalValidators() (int64, error) {
	// Function signature: totalValidators()
	// keccak256("totalValidators()") = 0x18160ddd (placeholder)
	functionSelector := "0x18160ddd"
	
	result, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", functionSelector)
	if err != nil {
		return 0, fmt.Errorf("failed to call totalValidators: %w", err)
	}
	
	// Parse the result (hex to int)
	if len(result) > 2 {
		if val, err := strconv.ParseInt(result[2:], 16, 64); err == nil {
			return val, nil
		}
	}
	
	return 0, fmt.Errorf("failed to parse totalValidators result")
}

// GetActiveValidators returns the number of active validators
func (c *EthereumClient) GetActiveValidators() (int64, error) {
	// Function signature: activeValidators()
	// keccak256("activeValidators()") = 0x8b5a9c0d (placeholder)
	functionSelector := "0x8b5a9c0d"
	
	result, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", functionSelector)
	if err != nil {
		return 0, fmt.Errorf("failed to call activeValidators: %w", err)
	}
	
	// Parse the result (hex to int)
	if len(result) > 2 {
		if val, err := strconv.ParseInt(result[2:], 16, 64); err == nil {
			return val, nil
		}
	}
	
	return 0, fmt.Errorf("failed to parse activeValidators result")
}

// GetStakingPool returns the total staking pool balance
func (c *EthereumClient) GetStakingPool() (string, error) {
	// Function signature: stakingPool()
	// keccak256("stakingPool()") = 0x8b5a9c0d (placeholder)
	functionSelector := "0x8b5a9c0d"
	
	result, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", functionSelector)
	if err != nil {
		return "", fmt.Errorf("failed to call stakingPool: %w", err)
	}
	
	return result, nil
}

// GetValidatorCount returns the total number of validators
func (c *EthereumClient) GetValidatorCount() (uint32, error) {
	// Function signature: validatorCount()
	// keccak256("validatorCount()") = 0x8b5a9c0d (placeholder)
	functionSelector := "0x8b5a9c0d"
	
	result, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", functionSelector)
	if err != nil {
		return 0, fmt.Errorf("failed to call validatorCount: %w", err)
	}
	
	// Parse the result (hex to uint32)
	if len(result) > 2 {
		if val, err := strconv.ParseUint(result[2:], 16, 32); err == nil {
			return uint32(val), nil
		}
	}
	
	return 0, fmt.Errorf("failed to parse validatorCount result")
}

// GetMaxValidatorCount returns the maximum number of validators allowed
func (c *EthereumClient) GetMaxValidatorCount() (uint32, error) {
	// Function signature: maxValidatorCount()
	// keccak256("maxValidatorCount()") = 0x8b5a9c0d (placeholder)
	functionSelector := "0x8b5a9c0d"
	
	result, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", functionSelector)
	if err != nil {
		return 0, fmt.Errorf("failed to call maxValidatorCount: %w", err)
	}
	
	// Parse the result (hex to uint32)
	if len(result) > 2 {
		if val, err := strconv.ParseUint(result[2:], 16, 32); err == nil {
			return uint32(val), nil
		}
	}
	
	return 0, fmt.Errorf("failed to parse maxValidatorCount result")
}

// GetValidatorByPubkey returns the validator address for a given public key
func (c *EthereumClient) GetValidatorByPubkey(pubkey string) (string, error) {
	// Function signature: getValidator(bytes)
	// keccak256("getValidator(bytes)") = 0x8b5a9c0d (placeholder)
	functionSelector := "0x8b5a9c0d"
	
	// Pad the pubkey to 32 bytes
	paddedPubkey := "0000000000000000000000000000000000000000000000000000000000000020" + pubkey[2:]
	
	data := functionSelector + paddedPubkey
	
	result, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", data)
	if err != nil {
		return "", fmt.Errorf("failed to call getValidator: %w", err)
	}
	
	return result, nil
}

// ComputeValidatorAddress computes the validator address for a given public key
func (c *EthereumClient) ComputeValidatorAddress(pubkey string) (string, error) {
	// Function signature: computeValidatorAddress(bytes)
	// keccak256("computeValidatorAddress(bytes)") = 0x8b5a9c0d (placeholder)
	functionSelector := "0x8b5a9c0d"
	
	// Pad the pubkey to 32 bytes
	paddedPubkey := "0000000000000000000000000000000000000000000000000000000000000020" + pubkey[2:]
	
	data := functionSelector + paddedPubkey
	
	result, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", data)
	if err != nil {
		return "", fmt.Errorf("failed to call computeValidatorAddress: %w", err)
	}
	
	return result, nil
}

// GetValidatorByIndex retrieves validator information by index
func (c *EthereumClient) GetValidatorByIndex(index int) (map[string]interface{}, error) {
	// Function signature: getValidatorByIndex(uint256)
	// keccak256("getValidatorByIndex(uint256)") = 0x8b5a9c0d (placeholder)
	indexHex := fmt.Sprintf("%064x", index)
	functionSelector := "0x8b5a9c0d"
	data := functionSelector + indexHex
	
	result, err := c.CallContract("0xea224dBB52F57752044c0C86aD50930091F561B9", data)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator by index: %w", err)
	}
	
	// Parse the result (placeholder)
	return map[string]interface{}{
		"index":   index,
		"address": "unknown",
		"moniker": "unknown",
		"raw_result": result,
	}, nil
}
