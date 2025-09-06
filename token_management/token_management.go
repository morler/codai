package token_management

import (
	"encoding/json"
	"fmt"
	"github.com/meysamhadeli/codai/constants/lipgloss"
	"github.com/meysamhadeli/codai/embed_data"
	"github.com/meysamhadeli/codai/token_management/contracts"
	"log"
	"strings"
)

// TokenManager implementation
type tokenManager struct {
	usedToken       int
	usedInputToken  int
	usedOutputToken int
}

type details struct {
	MaxTokens               int     `json:"max_tokens"`
	MaxInputTokens          int     `json:"max_input_tokens"`
	MaxOutputTokens         int     `json:"max_output_tokens"`
	InputCostPerMillionTokens       float64 `json:"input_cost_per_million_tokens,omitempty"`
	OutputCostPerMillionTokens      float64 `json:"output_cost_per_million_tokens,omitempty"`
	CacheReadInputMillionTokenCost  float64 `json:"cache_read_input_million_token_cost,omitempty"`
	Mode                    string  `json:"mode"`
	SupportsFunctionCalling bool    `json:"supports_function_calling,omitempty"`
}

type Models struct {
	ModelDetails map[string]details `json:"models"`
}

// NewTokenManager creates a new token manager
func NewTokenManager() contracts.ITokenManagement {
	return &tokenManager{
		usedToken:       0,
		usedInputToken:  0,
		usedOutputToken: 0,
	}
}

// UsedTokens accumulates the token count for the session.
func (tm *tokenManager) UsedTokens(inputToken int, outputToken int) {
	tm.usedInputToken += inputToken
	tm.usedOutputToken += outputToken
	tm.usedToken += inputToken + outputToken
}

func (tm *tokenManager) DisplayTokens(chatProviderName string, chatModel string) {

	cost := tm.CalculateCost(chatProviderName, chatModel, tm.usedInputToken, tm.usedOutputToken)

	tokenInfo := fmt.Sprintf("Token Used: %s - Cost: %s $ - Chat Model: %s", fmt.Sprint(tm.usedToken), fmt.Sprintf("%.6f", cost), chatModel)

	tokenBox := lipgloss.BoxStyle.Render(tokenInfo)
	fmt.Println(tokenBox)
}

func (tm *tokenManager) DisplayLiveTokens(chatProviderName string, chatModel string) {
	cost := tm.CalculateCost(chatProviderName, chatModel, tm.usedInputToken, tm.usedOutputToken)
	
	// 使用\r清除当前行并重新打印，实现实时更新效果
	fmt.Printf("\rToken Used: %d - Cost: $%.6f - Model: %s", tm.usedToken, cost, chatModel)
}

func (tm *tokenManager) DisplayLiveTokensWithPreview(chatProviderName string, chatModel string, previewInput int, previewOutput int) {
	// 显示当前累计的token + 本次预览的token
	totalInput := tm.usedInputToken + previewInput
	totalOutput := tm.usedOutputToken + previewOutput
	totalTokens := tm.usedToken + previewInput + previewOutput
	
	cost := tm.CalculateCost(chatProviderName, chatModel, totalInput, totalOutput)
	
	// 使用\r清除当前行并重新打印，实现实时更新效果
	fmt.Printf("\rToken Used: %d - Cost: $%.6f - Model: %s", totalTokens, cost, chatModel)
}

// DisplayTokenUsage shows token usage with additional context about the request
func (tm *tokenManager) DisplayTokenUsage(chatProviderName string, chatModel string, addedInputTokens int, addedOutputTokens int) {
	oldTotal := tm.usedToken
	oldInput := tm.usedInputToken
	oldOutput := tm.usedOutputToken
	oldCost := tm.CalculateCost(chatProviderName, chatModel, oldInput, oldOutput)
	
	// 计算新增cost
	newCost := tm.CalculateCost(chatProviderName, chatModel, oldInput+addedInputTokens, oldOutput+addedOutputTokens)
	
	if oldTotal > 0 && addedInputTokens+addedOutputTokens > 0 {
		fmt.Printf("\r[Tokens: +%d input / +%d output = +%d total]  ", 
			addedInputTokens, addedOutputTokens, addedInputTokens+addedOutputTokens)
		if newCost > oldCost {
			fmt.Printf("[Cost: +$%.6f]  ", newCost-oldCost)
		}
		fmt.Print("\n")
	}
}

func (tm *tokenManager) GetCurrentTokenUsage() (total int, input int, output int) {
	return tm.usedToken, tm.usedInputToken, tm.usedOutputToken
}

func (tm *tokenManager) ClearToken() {
	tm.usedToken = 0
	tm.usedInputToken = 0
	tm.usedOutputToken = 0
}

func (tm *tokenManager) CalculateCost(providerName string, modelName string, inputToken int, outputToken int) float64 {
	modelDetails, err := getModelDetails(providerName, modelName)
	if err != nil {
		return 0
	}
	// Calculate cost for input tokens (convert from per-million to actual cost)
	inputCost := float64(inputToken) * modelDetails.InputCostPerMillionTokens / 1000000.0

	// Calculate cost for output tokens (convert from per-million to actual cost)
	outputCost := float64(outputToken) * modelDetails.OutputCostPerMillionTokens / 1000000.0

	// Total cost
	totalCost := inputCost + outputCost

	return totalCost
}

func getModelDetails(providerName string, modelName string) (details, error) {

	providerName = strings.ToLower(providerName)
	modelName = strings.ToLower(modelName)

	if strings.HasPrefix(providerName, "azure") {
		modelName = "azure/" + modelName
	}

	// Initialize the Models struct to hold parsed JSON data
	models := Models{
		ModelDetails: make(map[string]details),
	}

	// Unmarshal the JSON data from the embedded file
	err := json.Unmarshal(embed_data.ModelDetails, &models)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return details{}, err
	}

	// Look up the model by name
	model, exists := models.ModelDetails[modelName]
	if !exists {
		return details{}, fmt.Errorf("model details price with name '%s' not found for provider '%s'", modelName, providerName)
	}

	return model, nil
}
