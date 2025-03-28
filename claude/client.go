package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
)

const (
	ClaudeAPIBaseURL = "https://api.anthropic.com/v1"
)

// Client represents a client for interacting with Claude API
type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new Claude API client
func NewClient() *Client {
	return &Client{
		APIKey:     os.Getenv("CLAUDE_API_KEY"),
		HTTPClient: &http.Client{},
	}
}

// ReviewCodeRequest represents a request to review code
type ReviewCodeRequest struct {
	Model     string   `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int      `json:"max_tokens"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ReviewCodeResponse represents a response from the code review
type ReviewCodeResponse struct {
	Content []MessageContent `json:"content"`
}

// MessageContent represents the content of a message
type MessageContent struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
}

// ReviewPullRequest reviews a pull request and returns detailed feedback
func (c *Client) ReviewPullRequest(diff string) (string, error) {
	log.Debug("Preparing pull request review request")
	
	prompt := fmt.Sprintf(`
You are an expert code reviewer examining a GitHub pull request. 
Please provide detailed, constructive feedback on this code. 
Focus on:

1. Potential bugs, edge cases, or performance issues
2. Code structure and organization
3. Readability and maintainability
4. Security vulnerabilities
5. Adherence to best practices and design patterns

For each issue found, include:
- The exact line numbers
- What the problem is
- Why it's a concern
- A suggested improvement

Here is the diff to review:

%s
`, diff)

	reqBody := ReviewCodeRequest{
		Model: "claude-3-7-sonnet-20250219",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 4096,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	log.Debug("Sending request to Claude API", "url", ClaudeAPIBaseURL+"/messages")
	req, err := http.NewRequest("POST", ClaudeAPIBaseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Error("Claude API request failed", "error", err)
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Error("Claude API returned error", "status", resp.StatusCode, "response", string(bodyBytes))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	log.Debug("Processing Claude API response")
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		return "", fmt.Errorf("invalid response format")
	}

	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid content format")
	}

	text, ok := firstContent["text"].(string)
	if !ok {
		return "", fmt.Errorf("text field not found in content")
	}

	log.Debug("Successfully parsed Claude API response", "responseLength", len(text))
	return text, nil
}