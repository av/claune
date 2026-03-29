package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

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

func AnalyzeToolIntent(toolName, input string, c config.ClauneConfig) (string, error) {
	if !c.AI.Enabled {
		return "tool:start", nil
	}
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return "tool:start", fmt.Errorf("AI enabled but no ANTHROPIC_API_KEY found")
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

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "tool:start", fmt.Errorf("AI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "tool:start", fmt.Errorf("AI API returned status %d", resp.StatusCode)
	}

	respBytes, _ := io.ReadAll(resp.Body)
	var cr ClaudeResponse
	if err := json.Unmarshal(respBytes, &cr); err != nil {
		return "tool:start", fmt.Errorf("AI response parse failed: %w", err)
	}

	if len(cr.Content) > 0 {
		text := strings.ToLower(strings.TrimSpace(cr.Content[0].Text))
		if strings.Contains(text, "destructive") {
			return "tool:destructive", nil
		} else if strings.Contains(text, "readonly") {
			return "tool:readonly", nil
		}
	}
	return "tool:start", nil
}

func AnalyzeResponseSentiment(responseText string, c config.ClauneConfig) (string, string) {
	if !c.AI.Enabled {
		return "", ""
	}
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return "", ""
	}

	model := c.AI.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	prompt := fmt.Sprintf("Analyze this AI response for urgency or sentiment: %q. If it's a critical error or extremely urgent, reply with 'URGENT'. If it's very positive/successful, reply with 'SUCCESS'. Otherwise, reply 'NEUTRAL'.", responseText)
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

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		respBytes, _ := io.ReadAll(resp.Body)
		var cr ClaudeResponse
		if json.Unmarshal(respBytes, &cr) == nil && len(cr.Content) > 0 {
			text := strings.ToUpper(strings.TrimSpace(cr.Content[0].Text))
			if strings.Contains(text, "URGENT") {
				return "panic", "random" // override to panic with random strategy
			} else if strings.Contains(text, "SUCCESS") {
				return "tool:success", "random"
			}
		}
	}
	return "", ""
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
Current time: %s
Reply with ONLY valid JSON representing the updated configuration fields. Do not include markdown blocks. Example: {"mute": true, "mute_until": "2023-10-12T14:00:00Z", "volume": 0.5, "sounds": {"tool:start": ["file.wav"]}}`, c, prompt, time.Now().Format(time.RFC3339))

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
		if s, ok := updates["sounds"].(map[string]interface{}); ok {
			if c.Sounds == nil {
				c.Sounds = make(map[string]config.EventSoundConfig)
			}
			for k, v := range s {
				if vs, ok := v.(string); ok {
					c.Sounds[k] = config.EventSoundConfig{Paths: []string{vs}}
				} else if arr, ok := v.([]interface{}); ok {
					var paths []string
					for _, item := range arr {
						if str, ok := item.(string); ok {
							paths = append(paths, str)
						}
					}
					c.Sounds[k] = config.EventSoundConfig{Paths: paths}
				} else if obj, ok := v.(map[string]interface{}); ok {
					var esc config.EventSoundConfig
					if paths, ok := obj["paths"].([]interface{}); ok {
						for _, item := range paths {
							if str, ok := item.(string); ok {
								esc.Paths = append(esc.Paths, str)
							}
						}
					}
					if strategy, ok := obj["strategy"].(string); ok {
						esc.Strategy = strategy
					}
					c.Sounds[k] = esc
				}
			}
		}
		if mu, ok := updates["mute_until"].(string); ok {
			if t, err := time.Parse(time.RFC3339, mu); err == nil {
				c.MuteUntil = &t
				f := false
				c.Mute = &f // Ensure Mute is false so MuteUntil takes precedence
			}
		}
		return config.Save(*c)
	}
	return nil
}

func AutoMapSounds(dir string, c *config.ClauneConfig) error {
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

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			name := e.Name()
			if strings.HasSuffix(name, ".mp3") || strings.HasSuffix(name, ".wav") || strings.HasSuffix(name, ".ogg") {
				files = append(files, name)
			}
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no audio files found in %s", dir)
	}

	sysPrompt := fmt.Sprintf(`You are an AI configuring a sound application. Map the following audio files to appropriate events based on their names.
Available events: cli:start, tool:start, tool:success, tool:error, cli:done, build:success, test:fail, panic, warn.
Available files: %s.
Directory path: %s.
Return ONLY a valid JSON object mapping event strings to an object with "paths" (array of full absolute paths) and "strategy" ("random" or "round_robin").
Do not include markdown blocks. Example: {"tool:success": {"paths": ["/dir/yay.mp3"], "strategy": "random"}}`, strings.Join(files, ", "), dir)

	reqBody := ClaudeRequest{
		Model: model,
		Messages: []ClaudeMessage{
			{Role: "user", Content: sysPrompt},
		},
		MaxTokens: 500,
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
		var mapping map[string]config.EventSoundConfig
		if err := json.Unmarshal([]byte(text), &mapping); err != nil {
			return fmt.Errorf("failed to parse AI response: %w\nResponse: %s", err, text)
		}

		if c.Sounds == nil {
			c.Sounds = make(map[string]config.EventSoundConfig)
		}
		for k, v := range mapping {
			c.Sounds[k] = v
		}

		return config.Save(*c)
	}

	return fmt.Errorf("no response from AI")
}
