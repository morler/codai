package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/meysamhadeli/codai/providers/contracts"
	"github.com/meysamhadeli/codai/providers/models"
	ollama_models "github.com/meysamhadeli/codai/providers/ollama/models"
	contracts2 "github.com/meysamhadeli/codai/token_management/contracts"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// OllamaConfig implements the Provider interface for OpenAPI.
type OllamaConfig struct {
	BaseURL         string
	Model           string
	Temperature     *float32
	ReasoningEffort *string
	EncodingFormat  string
	MaxTokens       int
	TokenManagement contracts2.ITokenManagement
}

const (
	defaultBaseURL = "http://localhost:11434/api"
)

// NewOllamaChatProvider initializes a new OpenAPIProvider.
func NewOllamaChatProvider(config *OllamaConfig) contracts.IChatAIProvider {
	// Set default BaseURL if empty
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &OllamaConfig{
		BaseURL:         config.BaseURL,
		Model:           config.Model,
		Temperature:     config.Temperature,
		ReasoningEffort: config.ReasoningEffort,
		EncodingFormat:  config.EncodingFormat,
		MaxTokens:       config.MaxTokens,
		TokenManagement: config.TokenManagement,
	}
}

func (ollamaProvider *OllamaConfig) ChatCompletionRequest(ctx context.Context, userInput string, prompt string) <-chan models.StreamResponse {
	responseChan := make(chan models.StreamResponse)
	var markdownBuffer strings.Builder // Buffer to accumulate content until newline

	go func() {
		defer close(responseChan)

		// Prepare the request body
		reqBody := ollama_models.OllamaChatCompletionRequest{
			Model: ollamaProvider.Model,
			Messages: []ollama_models.Message{
				{Role: "system", Content: prompt},
				{Role: "user", Content: userInput},
			},
			Stream:      true,
			Temperature: ollamaProvider.Temperature,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			markdownBuffer.Reset()
			responseChan <- models.StreamResponse{Err: fmt.Errorf("error marshalling request body: %v", err)}
			return
		}

		// Create a new HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/chat", ollamaProvider.BaseURL), bytes.NewBuffer(jsonData))
		if err != nil {
			markdownBuffer.Reset()
			responseChan <- models.StreamResponse{Err: fmt.Errorf("error creating request: %v", err)}
			return
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			markdownBuffer.Reset()
			if errors.Is(ctx.Err(), context.Canceled) {
				responseChan <- models.StreamResponse{Err: fmt.Errorf("request canceled: %v", err)}
				return
			}
			responseChan <- models.StreamResponse{Err: fmt.Errorf("error sending request: %v", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			markdownBuffer.Reset()

			body, _ := ioutil.ReadAll(resp.Body)
			var apiError models.AIError
			if err := json.Unmarshal(body, &apiError); err != nil {
				responseChan <- models.StreamResponse{Err: fmt.Errorf("error parsing error response: %v", err)}
				return
			}

			responseChan <- models.StreamResponse{Err: fmt.Errorf("API request failed with status code '%d' - %s\n", resp.StatusCode, apiError.Error.Message)}
			return
		}

		reader := bufio.NewReader(resp.Body)

		// Stream processing
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				markdownBuffer.Reset()

				if err == io.EOF {
					break
				}
				responseChan <- models.StreamResponse{Err: fmt.Errorf("error reading stream: %v", err)}
				return
			}

			var response ollama_models.OllamaChatCompletionResponse
			if err := json.Unmarshal([]byte(line), &response); err != nil {
				markdownBuffer.Reset()

				responseChan <- models.StreamResponse{Err: fmt.Errorf("error unmarshalling chunk: %v", err)}
				return
			}

			if len(response.Message.Content) > 0 {
				content := response.Message.Content
				markdownBuffer.WriteString(content)

				// Send chunk if it contains a newline, and then reset the buffer
				if strings.Contains(content, "\n") {
					responseChan <- models.StreamResponse{Content: markdownBuffer.String()}
					markdownBuffer.Reset()
				}
			}

			// Check if the response is marked as done
			if response.Done {
				//	// Signal end of stream
				responseChan <- models.StreamResponse{Content: markdownBuffer.String()}
				responseChan <- models.StreamResponse{Done: true}

				// Count total tokens usage
				if response.PromptEvalCount > 0 {
					ollamaProvider.TokenManagement.UsedTokens(response.PromptEvalCount, response.EvalCount)
					// 显示本次使用的token统计
					fmt.Print("\n")
					ollamaProvider.TokenManagement.DisplayTokenUsage(
						"ollama",
						ollamaProvider.Model,
						response.PromptEvalCount,
						response.EvalCount,
					)
				}

				break
			}
		}

		// Send any remaining content in the buffer
		if markdownBuffer.Len() > 0 {
			responseChan <- models.StreamResponse{Content: markdownBuffer.String()}
		}
	}()

	return responseChan
}
