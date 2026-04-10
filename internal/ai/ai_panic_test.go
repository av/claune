package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/everlier/claune/internal/config"
)

func TestHandleNaturalLanguageConfigPanic(t *testing.T) {
	// Setup mock AI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": `{"sounds": {"cli:start": {"paths": ["/tmp/a.mp3"]}}}`,
				},
			},
		})
	}))
	defer server.Close()

	// c.Sounds is nil !
	c := &config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: true,
			APIKey:  "test",
			APIURL:  server.URL,
		},
	}

	err := HandleNaturalLanguageConfig("add a sound", c)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
