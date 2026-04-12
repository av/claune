package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"fmt"
	"os"
	"path/filepath"

	"github.com/everlier/claune/internal/config"
)

func createMockAnthropicServer(t *testing.T, expectedResponse string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("Expected path /v1/messages, got %s", r.URL.Path)
		}
		
		if r.Method != "POST" {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		
		resp := ClaudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: expectedResponse},
			},
		}
		
		json.NewEncoder(w).Encode(resp)
	}))
}

func mockConfig(serverURL string, t *testing.T) config.ClauneConfig {
	return config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: true,
			APIKey:  "test-key",
			APIURL:  serverURL,
			Model:   "claude-3-opus-test",
		},
	}
}

func TestAnalyzeToolIntent_Mock(t *testing.T) {
	expectedResponse := `destructive`
	server := createMockAnthropicServer(t, expectedResponse)
	defer server.Close()

	c := mockConfig(server.URL, t)
	intent, err := AnalyzeToolIntent("tool:success", "bash", "rm -rf /", c)

	if err != nil {
		t.Fatalf("AnalyzeToolIntent failed: %v", err)
	}

	if intent != "tool:destructive" {
		t.Errorf("Expected tool:destructive, got %s", intent)
	}
}

func TestAnalyzeResponseSentiment_Mock(t *testing.T) {
	expectedResponse := `SUCCESS`
	server := createMockAnthropicServer(t, expectedResponse)
	defer server.Close()

	c := mockConfig(server.URL, t)
	sentiment, strategy, err := AnalyzeResponseSentiment("Great job!", c)

	if err != nil {
		t.Fatalf("AnalyzeResponseSentiment failed: %v", err)
	}

	if sentiment != "tool:success" {
		t.Errorf("Expected sentiment tool:success, got %s", sentiment)
	}
	
	if strategy != "random" {
		t.Errorf("Expected strategy random, got %s", strategy)
	}
}

func TestHandleNaturalLanguageConfig_Mock(t *testing.T) {
	expectedJSON := `{
		"sounds": {
			"startup": {
				"paths": ["/path/to/startup.mp3"]
			}
		}
	}`
	server := createMockAnthropicServer(t, expectedJSON)
	defer server.Close()

	c := mockConfig(server.URL, t)
	c.Sounds = make(map[string]config.EventSoundConfig)
	
	err := HandleNaturalLanguageConfig("Change startup sound to /path/to/startup.mp3", &c)

	if err != nil {
		t.Fatalf("HandleNaturalLanguageConfig failed: %v", err)
	}

	if s, ok := c.Sounds["startup"]; !ok || len(s.Paths) == 0 || s.Paths[0] != "/path/to/startup.mp3" {
		t.Errorf("Config not updated correctly, got %+v", c.Sounds["startup"])
	}
}

func TestAutoMapSounds_Mock(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dir := t.TempDir()
	filePath := filepath.Join(dir, "victory.mp3")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	file.Close()

	expectedJSON := fmt.Sprintf(`{
		"tool:success": {
			"paths": ["%s"],
			"strategy": "random"
		}
	}`, filepath.ToSlash(filePath)) // Ensure the path string has correct slashes

	server := createMockAnthropicServer(t, expectedJSON)
	defer server.Close()

	c := mockConfig(server.URL, t)
	mapping, err := AutoMapSounds(dir, &c)

	if err != nil {
		t.Fatalf("AutoMapSounds failed: %v", err)
	}

	if event, ok := mapping["tool:success"]; !ok || len(event.Paths) == 0 || filepath.Base(event.Paths[0]) != "victory.mp3" {
		t.Errorf("Sound not mapped correctly, got %+v", mapping["tool:success"])
	}
}

func TestDiagnoseInstallFailure_Mock(t *testing.T) {
	expectedResponse := `Check your permissions`
	server := createMockAnthropicServer(t, expectedResponse)
	defer server.Close()

	c := mockConfig(server.URL, t)
	
	diagnosis := DiagnoseInstallFailure(fmt.Errorf("test err"), c)

	if diagnosis != "Check your permissions" {
		t.Errorf("Expected 'Check your permissions', got '%s'", diagnosis)
	}
}

func TestGuessEventForSound_Mock(t *testing.T) {
	expectedResponse := `tool:start`
	server := createMockAnthropicServer(t, expectedResponse)
	defer server.Close()

	c := mockConfig(server.URL, t)
	
	event, err := GuessEventForSound("http://example.com", "startup.mp3", c)

	if err != nil {
		t.Fatalf("GuessEventForSound failed: %v", err)
	}

	if event != "tool:start" {
		t.Errorf("Expected 'tool:start', got '%s'", event)
	}
}
