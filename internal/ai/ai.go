package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/everlier/claune/internal/config"
)

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeRequest struct {
	Model     string          `json:"model"`
	Messages  []ClaudeMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func AnalyzeToolIntent(toolName, input string, c config.ClauneConfig) string {
	if !c.AI.Enabled {
		return "tool:start"
	}
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return "tool:start"
	}

	model := c.AI.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	prompt := fmt.Sprintf("Analyze this tool call: %s with input: %s. Reply with ONE WORD ONLY: 'destructive' if it mutates data, deletes files, or executes arbitrary code. 'readonly' if it just reads data.", toolName, input)
	reqBody := ClaudeRequest{
		Model: model,
		Messages: []ClaudeMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 10,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "tool:start"
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	var cr ClaudeResponse
	json.Unmarshal(respBytes, &cr)

	if len(cr.Content) > 0 {
		text := strings.ToLower(strings.TrimSpace(cr.Content[0].Text))
		if strings.Contains(text, "destructive") {
			return "tool:start" // could return a different sound if we want
		}
	}
	return "tool:start"
}

func HandleNaturalLanguageConfig(prompt string, c *config.ClauneConfig) error {
	if !c.AI.Enabled {
		return fmt.Errorf("AI is disabled")
	}
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return fmt.Errorf("no ANTHROPIC_API_KEY")
	}

	model := c.AI.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	sysPrompt := fmt.Sprintf(`You are configuring Claune, an audio tool. Current config: %+v.
User prompt: %s
Reply with ONLY valid JSON representing the updated configuration fields. Do not include markdown blocks. Example: {"mute": true, "volume": 0.5}`, c, prompt)

	reqBody := ClaudeRequest{
		Model: model,
		Messages: []ClaudeMessage{
			{Role: "user", Content: sysPrompt},
		},
		MaxTokens: 200,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	var cr ClaudeResponse
	if err := json.Unmarshal(respBytes, &cr); err != nil {
		return err
	}

	if len(cr.Content) > 0 {
		text := strings.TrimSpace(cr.Content[0].Text)
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSuffix(text, "```")
		var updates map[string]interface{}
		if err := json.Unmarshal([]byte(text), &updates); err != nil {
			return err
		}
		
		// apply updates
		if m, ok := updates["mute"].(bool); ok {
			c.Mute = &m
		}
		if v, ok := updates["volume"].(float64); ok {
			c.Volume = &v
		}
		return config.Save(*c)
	}
	return nil
}
