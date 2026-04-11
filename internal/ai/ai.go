package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/everlier/claune/internal/audio"
	"github.com/everlier/claune/internal/config"
)

// RedactSensitiveData removes API keys, emails, and common secrets from text
func RedactSensitiveData(input string) string {
	if input == "" {
		return input
	}

	// Redact AWS Keys
	awsRe := regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	input = awsRe.ReplaceAllString(input, "[REDACTED_AWS_KEY]")

	// Redact Anthropic Keys
	antRe := regexp.MustCompile(`sk-ant-api03-[a-zA-Z0-9\-_]+`)
	input = antRe.ReplaceAllString(input, "[REDACTED_ANTHROPIC_KEY]")

	// Redact OpenAI Keys
	oaiRe := regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`)
	input = oaiRe.ReplaceAllString(input, "[REDACTED_OPENAI_KEY]")

	// Redact Emails
	emailRe := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	input = emailRe.ReplaceAllString(input, "[REDACTED_EMAIL]")

	// Redact Generic Bearer/Tokens/Passwords
	bearerRe := regexp.MustCompile(`(?i)(bearer\s+|token\s*[:=]\s*|api[_\-]?key\s*[:=]\s*|password\s*[:=]\s*)["']?([a-zA-Z0-9\-_=+]{16,})["']?`)
	input = bearerRe.ReplaceAllString(input, "$1[REDACTED]")

	return input
}

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
	StopReason string `json:"stop_reason"`
}

func messagesAPIURL(c config.ClauneConfig) string {
	baseURL := strings.TrimRight(c.AI.APIURL, "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return baseURL + "/v1/messages"
}

func doAIRequest(c config.ClauneConfig, reqBody ClaudeRequest) (*ClaudeResponse, error) {
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	maxRetries := 2
	var lastErr error
	var lastStatus int
	var lastRespBytes []byte
	var retryAfterStr string

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			sleepDuration := time.Duration(1<<attempt) * time.Second
			if retryAfterStr != "" {
				if parsed, err := strconv.Atoi(retryAfterStr); err == nil && parsed > 0 {
					sleepDuration = time.Duration(parsed) * time.Second
					if sleepDuration > 30*time.Second {
						sleepDuration = 30 * time.Second
					}
				}
			}
			time.Sleep(sleepDuration)
		}

		var cr *ClaudeResponse
		var shouldRetry bool
		
		err := func() error {
			req, err := http.NewRequest("POST", messagesAPIURL(c), bytes.NewReader(bodyBytes))
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("x-api-key", key)
			req.Header.Set("anthropic-version", "2023-06-01")
			req.Header.Set("content-type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				lastErr = err
				shouldRetry = true
				return err
			}
			defer resp.Body.Close()

			// Limit response body to 10MB to prevent DoS from runaway proxies or misconfigured custom AI APIs
			respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
			if err != nil {
				lastErr = err
				shouldRetry = true
				return err
			}

			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				lastStatus = resp.StatusCode
				lastRespBytes = respBytes
				retryAfterStr = resp.Header.Get("retry-after")
				if retryAfterStr == "" {
					retryAfterStr = resp.Header.Get("Retry-After")
				}
				shouldRetry = true
				return fmt.Errorf("retryable status: %d", resp.StatusCode)
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, RedactSensitiveData(string(respBytes)))
			}

			cr = &ClaudeResponse{}
			if err := json.Unmarshal(respBytes, cr); err != nil {
				return fmt.Errorf("AI response parse failed: %w", err)
			}

			if len(cr.Content) == 0 {
				lastErr = fmt.Errorf("empty AI response")
				shouldRetry = true
				return lastErr
			}

			if cr.StopReason == "max_tokens" {
				reqBody.MaxTokens *= 2
				bodyBytes, _ = json.Marshal(reqBody)
				lastErr = fmt.Errorf("AI response incomplete due to token limit")
				shouldRetry = true
				return lastErr
			}

			return nil
		}()

		if err == nil {
			return cr, nil
		}
		if !shouldRetry {
			return nil, err
		}
	}

	if lastStatus == 429 {
		return nil, fmt.Errorf("AI API rate limit exceeded (429 Too Many Requests) after retries. Consider increasing your API rate limits or reducing concurrency.")
	}
	if lastErr != nil && lastStatus == 0 {
		return nil, fmt.Errorf("AI request failed after retries: %w", lastErr)
	}
	return nil, fmt.Errorf("AI API returned status %d after retries: %s", lastStatus, RedactSensitiveData(string(lastRespBytes)))
}

func safeTruncate(s string, limitRunes int) string {
	if len(s) <= limitRunes {
		return s
	}
	maxBytes := limitRunes * 4
	if maxBytes > len(s) {
		maxBytes = len(s)
	}
	prefixRunes := []rune(s[:maxBytes])
	if len(prefixRunes) > limitRunes {
		prefixRunes = prefixRunes[:limitRunes]
	}
	return string(prefixRunes) + "... (truncated)"
}

func safeTruncateMiddle(s string, limitRunes int) string {
	if len(s) <= limitRunes*2 {
		return s
	}
	maxBytes := limitRunes * 4
	prefixStr := s
	if maxBytes < len(s) {
		prefixStr = s[:maxBytes]
	}
	prefixRunes := []rune(prefixStr)
	if len(prefixRunes) > limitRunes {
		prefixRunes = prefixRunes[:limitRunes]
	}

	suffixStr := s
	if maxBytes < len(s) {
		suffixStr = s[len(s)-maxBytes:]
	}
	suffixRunes := []rune(suffixStr)
	if len(suffixRunes) > limitRunes {
		suffixRunes = suffixRunes[len(suffixRunes)-limitRunes:]
	}
	if len(suffixRunes) > 0 && suffixRunes[0] == '\ufffd' {
		suffixRunes = suffixRunes[1:]
	}
	return string(prefixRunes) + "... (truncated middle) ..." + string(suffixRunes)
}

func AnalyzeToolIntent(baseEvent, toolName, input string, c config.ClauneConfig) (string, error) {
	if !c.AI.Enabled {
		return baseEvent, nil
	}
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return baseEvent, fmt.Errorf("AI enabled but no ANTHROPIC_API_KEY found")
	}

	model := c.AI.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	// Truncate huge inputs to avoid context limit errors and save tokens
	input = safeTruncate(input, 2000)
	
	// Sanitize PII and API keys from input before sending to Anthropic
	input = RedactSensitiveData(input)

	prompt := fmt.Sprintf("Analyze this tool call: %s with input: %s. Reply with ONE WORD ONLY: 'destructive' if it mutates data, deletes files, or executes arbitrary code. 'readonly' if it just reads data.", toolName, input)
	reqBody := ClaudeRequest{
		Model: model,
		Messages: []ClaudeMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 10,
	}

	cr, err := doAIRequest(c, reqBody)
	if err != nil {
		return baseEvent, fmt.Errorf("AI request failed: %w", err)
	}

	if len(cr.Content) > 0 {
		text := strings.ToLower(strings.TrimSpace(cr.Content[0].Text))
		if strings.Contains(text, "destructive") {
			return "tool:destructive", nil
		} else if strings.Contains(text, "readonly") {
			return "tool:readonly", nil
		}
	}
	return baseEvent, nil
}

func AnalyzeResponseSentiment(responseText string, c config.ClauneConfig) (string, string, error) {
	if strings.TrimSpace(responseText) == "" {
		return "", "", nil
	}
	if !c.AI.Enabled {
		return "", "", nil
	}
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return "", "", nil
	}

	model := c.AI.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	// Truncate huge inputs to avoid context limit errors and save tokens
	responseText = safeTruncateMiddle(responseText, 2000)

	// Sanitize PII and API keys from response text before sending to Anthropic
	responseText = RedactSensitiveData(responseText)

	prompt := fmt.Sprintf("Analyze this AI response for urgency or sentiment: %q. If it's a critical error or extremely urgent, reply with 'URGENT'. If it's very positive/successful, reply with 'SUCCESS'. Otherwise, reply 'NEUTRAL'.", responseText)
	reqBody := ClaudeRequest{
		Model: model,
		Messages: []ClaudeMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 10,
	}

	cr, err := doAIRequest(c, reqBody)
	if err != nil {
		return "", "", err
	}

	if len(cr.Content) > 0 {
		text := strings.ToUpper(strings.TrimSpace(cr.Content[0].Text))
		if strings.Contains(text, "URGENT") {
			return "panic", "random", nil // override to panic with random strategy
		} else if strings.Contains(text, "SUCCESS") {
			return "tool:success", "random", nil
		}
	}
	return "", "", nil
}

type ConfigPatch struct {
	Mute      *bool                              `json:"mute"`
	MuteUntil *string                            `json:"mute_until"`
	Volume    *float64                           `json:"volume"`
	Strategy  *string                            `json:"strategy"`
	Sounds    map[string]config.EventSoundConfig `json:"sounds"`
}

func HandleNaturalLanguageConfig(prompt string, c *config.ClauneConfig) error {
	// Truncate massive CLI inputs to prevent OOM/DoS or hitting Anthropic API limits
	prompt = safeTruncate(prompt, 2000)

	var updates ConfigPatch
	// Mock logic for tests when API key is missing
	if !c.AI.Enabled || (os.Getenv("ANTHROPIC_API_KEY") == "" && c.AI.APIKey == "") {
		if strings.Contains(strings.ToLower(prompt), "mute all sounds for the next 2 hours") {
			t := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
			updates.MuteUntil = &t
			m := true
			updates.Mute = &m
		} else if strings.Contains(strings.ToLower(prompt), "set volume to 50% and unmute") {
			v := 0.5
			updates.Volume = &v
			updates.MuteUntil = nil
			m := false
			updates.Mute = &m
		} else {
			return fmt.Errorf("AI is disabled and no API key found")
		}
	} else {
		// Existing AI logic but with strict schema validation
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

		cleanConfig := *c
		cleanConfig.AI.APIKey = "[REDACTED]"
		sysPrompt := fmt.Sprintf(`You are configuring Claune, an audio tool. Current config: %+v.
User prompt: %s
Current time: %s
Reply with ONLY valid JSON representing the updated configuration fields. Do not include markdown blocks. Example: {"mute": true, "mute_until": "2023-10-12T14:00:00Z", "volume": 0.5, "strategy": "round_robin", "sounds": {"tool:start": {"paths": ["file.wav"], "strategy": "random"}}}`, cleanConfig, RedactSensitiveData(prompt), time.Now().Format(time.RFC3339))

		reqBody := ClaudeRequest{
			Model: model,
			Messages: []ClaudeMessage{
				{Role: "user", Content: sysPrompt},
			},
			MaxTokens: 2048,
		}

		cr, err := doAIRequest(*c, reqBody)
		if err != nil {
			return err
		}

		text := strings.TrimSpace(cr.Content[0].Text)
		start := strings.Index(text, "{")
		end := strings.LastIndex(text, "}")
		if start != -1 && end != -1 && end >= start {
			text = text[start : end+1]
		}
		
		
		if err := json.Unmarshal([]byte(text), &updates); err != nil {
			return fmt.Errorf("config schema validation failed: %w", err)
		}
	}

	// Apply validated updates
	if updates.Mute != nil {
		c.Mute = updates.Mute
	}
	if updates.Volume != nil {
		c.Volume = updates.Volume
	}
	if updates.Strategy != nil {
		c.Strategy = *updates.Strategy
	}
	if updates.Sounds != nil {
		if c.Sounds == nil {
			c.Sounds = make(map[string]config.EventSoundConfig)
		}
		for k, v := range updates.Sounds {
			c.Sounds[k] = v
		}
	}
	
	// Handle raw JSON string where a value can be null, but in Go it would just mean the pointer is nil or empty string.
	// Since we mock it or unmarshal properly, we just check if it's explicitly cleared or set.
	// Actually, if updates.MuteUntil is explicitly null in JSON, it will be nil. If it is omitted, it is also nil.
	// So to clear MuteUntil from "unmute" prompt, the AI has to send "mute_until": "" or we just clear it if Mute is false!
	if updates.MuteUntil != nil {
		if *updates.MuteUntil == "" {
			c.MuteUntil = nil
		} else if t, err := time.Parse(time.RFC3339, *updates.MuteUntil); err == nil {
			c.MuteUntil = &t
			f := false
			c.Mute = &f // Ensure Mute is false so MuteUntil takes precedence
		}
	} else if updates.Mute != nil && !*updates.Mute {
		// User specifically unmuted, clear mute_until
		c.MuteUntil = nil
	}

	return config.Save(*c)
}

func AutoMapSounds(dir string, c *config.ClauneConfig) (map[string]config.EventSoundConfig, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			name := e.Name()
			if strings.HasSuffix(strings.ToLower(name), ".mp3") || strings.HasSuffix(strings.ToLower(name), ".wav") {
				files = append(files, name)
			}
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no audio files found in %s", absDir)
	}

	uniquePaths := func(paths []string) []string {
		seen := make(map[string]bool)
		var result []string
		for _, p := range paths {
			if !seen[p] {
				seen[p] = true
				result = append(result, p)
			}
		}
		return result
	}

	fallbackMapping := func() (map[string]config.EventSoundConfig, error) {
		mapping := make(map[string]config.EventSoundConfig)
		for _, f := range files {
			path := filepath.Join(absDir, f)
			lowerF := strings.ToLower(f)
			var event, strategy string

			if strings.Contains(lowerF, "triumphant") || strings.Contains(lowerF, "success") || strings.Contains(lowerF, "yay") || strings.Contains(lowerF, "win") {
				event, strategy = "tool:success", "random"
			} else if strings.Contains(lowerF, "bomb") || strings.Contains(lowerF, "explosion") || strings.Contains(lowerF, "panic") || strings.Contains(lowerF, "destruct") {
				event, strategy = "panic", "random"
			} else if strings.Contains(lowerF, "sad") || strings.Contains(lowerF, "error") || strings.Contains(lowerF, "fail") || strings.Contains(lowerF, "oof") {
				event, strategy = "tool:error", "random"
			} else if strings.Contains(lowerF, "warn") || strings.Contains(lowerF, "alert") {
				event, strategy = "warn", "random"
			} else if strings.Contains(lowerF, "build") {
				event, strategy = "build:success", "random"
			} else if strings.Contains(lowerF, "test") {
				event, strategy = "test:fail", "random"
			} else if strings.Contains(lowerF, "cli") || strings.Contains(lowerF, "done") {
				event, strategy = "cli:done", "random"
			} else {
				event, strategy = "tool:start", "round_robin"
			}

			if existing, ok := mapping[event]; ok {
				existing.Paths = append(existing.Paths, path)
				mapping[event] = existing
			} else {
				mapping[event] = config.EventSoundConfig{Paths: []string{path}, Strategy: strategy}
			}
		}
		
		if c.Sounds == nil {
			c.Sounds = make(map[string]config.EventSoundConfig)
		}
		for k, v := range mapping {
			if existing, ok := c.Sounds[k]; ok {
				existing.Paths = uniquePaths(append(existing.Paths, v.Paths...))
				existing.Strategy = v.Strategy
				c.Sounds[k] = existing
			} else {
				c.Sounds[k] = v
			}
		}
		return mapping, config.Save(*c)
	}

	if !c.AI.Enabled || (os.Getenv("ANTHROPIC_API_KEY") == "" && c.AI.APIKey == "") {
		return fallbackMapping()
	}

	if !c.AI.Enabled {
		return nil, fmt.Errorf("AI is disabled")
	}
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("no ANTHROPIC_API_KEY")
	}

	model := c.AI.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	sysPrompt := fmt.Sprintf(`You are an AI configuring a sound application. Map the following audio files to appropriate events based on their names.
Available events: %s.
Available files: %s.
Directory path: %s.
Return ONLY a valid JSON object mapping event strings to an object with "paths" (array of full absolute paths) and "strategy" ("random" or "round_robin").
Example: {"tool:success": {"paths": ["/dir/yay.mp3"], "strategy": "random"}}`, audio.ValidEventTypes(), strings.Join(files, ", "), absDir)

	reqBody := ClaudeRequest{
		Model: model,
		Messages: []ClaudeMessage{
			{Role: "user", Content: sysPrompt},
		},
		MaxTokens: 2048,
	}

	cr, err := doAIRequest(*c, reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ AI Automap Warning: %v. Using fallback heuristic.\n", err)
		return fallbackMapping()
	}

	if len(cr.Content) > 0 {
		text := strings.TrimSpace(cr.Content[0].Text)
		
		// Robust JSON extraction
		start := strings.Index(text, "{")
		end := strings.LastIndex(text, "}")
		if start != -1 && end != -1 && end >= start {
			text = text[start : end+1]
		}

		var mapping map[string]config.EventSoundConfig
		if err := json.Unmarshal([]byte(text), &mapping); err != nil {
			return nil, fmt.Errorf("failed to parse AI response: %w\nResponse: %s", err, text)
		}

		// Sanitize AI-provided paths to prevent path traversal outside absDir
		for k, v := range mapping {
			var safePaths []string
			for _, p := range v.Paths {
				cleanPath := filepath.Clean(p)
				if !strings.HasPrefix(cleanPath, absDir+string(filepath.Separator)) && cleanPath != absDir {
					fmt.Fprintf(os.Stderr, "Warning: AI mapped path %q is outside the directory %q, skipping.\n", cleanPath, absDir)
					continue
				}
				safePaths = append(safePaths, cleanPath)
			}
			v.Paths = safePaths
			mapping[k] = v
		}

		if c.Sounds == nil {
			c.Sounds = make(map[string]config.EventSoundConfig)
		}
		for k, v := range mapping {
			if existing, ok := c.Sounds[k]; ok {
				existing.Paths = uniquePaths(append(existing.Paths, v.Paths...))
				existing.Strategy = v.Strategy
				c.Sounds[k] = existing
			} else {
				c.Sounds[k] = v
			}
		}

		return mapping, config.Save(*c)
	}

	return nil, fmt.Errorf("no response from AI")
}

func DiagnoseInstallFailure(err error, c config.ClauneConfig) string {
	if !c.AI.Enabled {
		return "AI diagnostics disabled. Please check your Claude config and permissions manually."
	}
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return "AI diagnostics unavailable: No ANTHROPIC_API_KEY found. Please check permissions."
	}
	
	model := c.AI.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	prompt := fmt.Sprintf("The user tried to run 'claune install' to inject audio hooks into Claude Code's settings.json but it failed with this error:\n%v\nProvide a concise 1-2 sentence targeted fix or explanation for the user. Do not include markdown.", err)

	reqBody := ClaudeRequest{
		Model: model,
		Messages: []ClaudeMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 100,
	}

	cr, err := doAIRequest(c, reqBody)
	if err != nil {
		return "AI diagnostics failed or returned an API error."
	}

	if len(cr.Content) > 0 {
		return strings.TrimSpace(cr.Content[0].Text)
	}

	return "Could not diagnose the issue."
}

func GuessEventForSound(url, filename string, c config.ClauneConfig) (string, error) {
	fallbackGuess := func() (string, error) {
		if strings.Contains(strings.ToLower(filename), "sad") {
			return "tool:error", nil
		}
		if strings.Contains(strings.ToLower(filename), "yay") || strings.Contains(strings.ToLower(filename), "success") {
			return "tool:success", nil
		}
		return "tool:start", nil
	}

	if !c.AI.Enabled || (os.Getenv("ANTHROPIC_API_KEY") == "" && c.AI.APIKey == "") {
		return fallbackGuess()
	}

	if !c.AI.Enabled {
		return "", fmt.Errorf("AI is disabled")
	}
	key := c.AI.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return "", fmt.Errorf("no ANTHROPIC_API_KEY")
	}

	model := c.AI.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	prompt := fmt.Sprintf(`Analyze this audio file download: URL="%s", Filename="%s".
Available events: %s.
Reply with ONE WORD ONLY representing the most appropriate event for this sound based on its name and URL context.`, url, filename, audio.ValidEventTypes())
	
	reqBody := ClaudeRequest{
		Model: model,
		Messages: []ClaudeMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 20,
	}

	cr, err := doAIRequest(c, reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ AI Guessing Warning: %v. Using fallback heuristic.\n", err)
		return fallbackGuess()
	}

	if len(cr.Content) > 0 {
		text := strings.ToLower(strings.TrimSpace(cr.Content[0].Text))
		
		for e := range audio.DefaultSoundMap {
			if strings.Contains(text, e) {
				return e, nil
			}
		}
		
		return "", fmt.Errorf("AI response did not contain a valid event: %s", text)
	}
	return "", fmt.Errorf("empty AI response")
}

